package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/Yi-Jiahe/wireguard_on_demand/wireguard"
)

var (
	logger *slog.Logger
)

type Stack struct {
	ProjectName string
	StackName   string
	OrgName     string
}

type Plugin struct {
	Name    string
	Version string
}

func main() {
	logger = slog.Default()

	args := os.Args[1:]

	if len(args) != 1 {
		logger.Error("unexpected arguments", "args", args)
		os.Exit(1)
	}

	destroy := false
	update := false
	if args[0] == "destroy" {
		destroy = true
	} else if args[0] == "update" {
		update = true
	} else {
		logger.Error("unexpected arguments", "args", args)
		os.Exit(1)
	}

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	stack := &Stack{
		ProjectName: "wireguard_on_demand",
		StackName:   "dev",
		OrgName:     "organization",
	}

	requiredPlugins := []Plugin{
		{
			Name:    "aws",
			Version: "v4.0.0",
		},
	}

	config := map[string]auto.ConfigValue{
		"aws:region": {
			Value: "ap-northeast-1", // Tokyo
		},
	}

	s, err := startStack(ctx, stack, requiredPlugins, config)
	if err != nil {
		panic(err)
	}

	if destroy {
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)
		_, err := s.Destroy(ctx, stdoutStreamer)
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	if update {
		stdoutStreamer := optup.ProgressStreams(os.Stdout)
		res, err := s.Up(ctx, stdoutStreamer)
		if err != nil {
			panic(err)
		}

		clientConfig := wireguard.Config{
			Interface: wireguard.Interface{
				PrivateKey: res.Outputs["client-private-key"].Value.(string),
				Address:    res.Outputs["client-address"].Value.(string),
				DNS:        "1.1.1.1",
			},
			Peers: []wireguard.Peer{
				{
					PublicKey:  res.Outputs["server-public-key"].Value.(string),
					Endpoint:   fmt.Sprintf("%s:%d", res.Outputs["server-endpoint"].Value.(string), int(res.Outputs["server-listen-port"].Value.(float64))),
					AllowedIps: []string{"0.0.0.0/0"},
				},
			},
		}

		fmt.Print(clientConfig.String())

		os.Exit(0)
	}
}

func startStack(ctx context.Context, stack *Stack, plugins []Plugin, config map[string]auto.ConfigValue) (auto.Stack, error) {
	projectName := stack.ProjectName
	stackName := auto.FullyQualifiedStackName(stack.OrgName, projectName, stack.StackName)

	s, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, wireguardHostDeployFunc)
	if err != nil {
		logger.Error("failed to create stack", "error", err, "stack", stackName)
		return s, err
	}
	logger.Info("created stack", "stack", stackName)

	w := s.Workspace()

	for _, plugin := range plugins {
		if err := w.InstallPlugin(ctx, plugin.Name, plugin.Version); err != nil {
			logger.Error("failed to install plugin", "plugin", plugin.Name, "version", plugin.Version, "error", err)
			return s, err
		}
	}
	logger.Info("installed plugins", "stack", stackName)

	for k, v := range config {
		s.SetConfig(ctx, k, v)
	}
	logger.Info("set config", "stack", stackName)

	_, err = s.Refresh(ctx)
	if err != nil {
		logger.Error("failed to refresh stack", "error", err, "stack", stackName)
		return s, err
	}
	logger.Info("refreshed stack", "stack", stackName)

	return s, nil
}

func wireguardHostDeployFunc(ctx *pulumi.Context) error {
	// Searching for the prefix list IDs isn't turning up any results for some reason
	instanceConnectPrefixListIds := pulumi.StringArray{
		pulumi.String("pl-012493c5f82b88e8e"), // com.amazonaws.ap-northeast-1.ec2-instance-connect
		pulumi.String("pl-08d491d20eebc3b95"), // com.amazonaws.ap-northeast-1.ipv6.ec2-instance-connect
	}

	sg, err := ec2.NewSecurityGroup(ctx, "wg-host", &ec2.SecurityGroupArgs{
		Description: pulumi.String("Allow UDP traffic for WireGuard"),
		Ingress: ec2.SecurityGroupIngressArray{
			// WireGuard Traffic
			ec2.SecurityGroupIngressArgs{
				Protocol: pulumi.String("udp"),
				FromPort: pulumi.Int(51820),
				ToPort:   pulumi.Int(51820),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
			},
			ec2.SecurityGroupIngressArgs{
				// SSH
				Protocol:      pulumi.String("tcp"),
				FromPort:      pulumi.Int(22),
				ToPort:        pulumi.Int(22),
				PrefixListIds: instanceConnectPrefixListIds,
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			ec2.SecurityGroupEgressArgs{
				Protocol: pulumi.String("-1"),
				FromPort: pulumi.Int(0),
				ToPort:   pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	ubuntu, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
		MostRecent: pulumi.BoolRef(true),
		Filters: []ec2.GetAmiFilter{
			{
				Name: "name",
				Values: []string{
					"ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*",
				},
			},
		},
		Owners: []string{"099720109477"}, // Amazon?
	})
	if err != nil {
		return err
	}

	userDataBytes, err := os.ReadFile("user_data.sh")
	if err != nil {
		return err
	}
	userData := string(userDataBytes)

	serverPrivateKey, serverPublicKey, err := wireguard.GenerateWireGuardKeyPair()
	if err != nil {
		return err
	}
	serverListenPort := 51820

	serverConfig := wireguard.Config{
		Interface: wireguard.Interface{
			PrivateKey: serverPrivateKey,
			ListenPort: serverListenPort,
		},
		Peers: []wireguard.Peer{},
	}

	clientPrivateKey, clientPublicKey, err := wireguard.GenerateWireGuardKeyPair()
	if err != nil {
		return err
	}
	clientAddress := "10.0.0.2/24"
	serverConfig.Peers = append(serverConfig.Peers, wireguard.Peer{
		PublicKey:  clientPublicKey,
		AllowedIps: []string{clientAddress},
	})

	userData = fmt.Sprintf(userData, serverConfig.String())
	fmt.Print(userData)

	wgHost, err := ec2.NewInstance(ctx, "wg-host", &ec2.InstanceArgs{
		Ami:          pulumi.String(ubuntu.Id),
		InstanceType: pulumi.String(ec2.InstanceType_T2_Micro),
		VpcSecurityGroupIds: pulumi.StringArray{
			sg.ID(),
		},
		UserData: pulumi.String(userData),
		Tags: pulumi.StringMap{
			"Project": pulumi.String("wireguard-on-demand"),
		},
	})
	if err != nil {
		return err
	}

	ctx.Export("server-endpoint", wgHost.PublicIp)
	ctx.Export("server-listen-port", pulumi.Int(serverListenPort))
	ctx.Export("server-public-key", pulumi.String(serverPublicKey))
	ctx.Export("client-private-key", pulumi.String(clientPrivateKey))
	ctx.Export("client-public-key", pulumi.String(clientPublicKey))
	ctx.Export("client-address", pulumi.String(clientAddress))

	return nil
}
