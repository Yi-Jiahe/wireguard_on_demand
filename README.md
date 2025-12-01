# Wireguard on Demand

The aim of this project is to enable the provisioning of self-hosted private tunnels.

The main rationale for self-hosting the solution is to remove reliance on VPN service providers and associated concerns.

Provisioning resources on demand allows for reduced long terms costs versus upfront VPN service plans.

The simplicity of the solution means that lacks features and reliability guarantees offered by VPN services apart from its intended usage.
One major flaw is the server availability.
The current implementation only supports AWS EC2 regions.

## Usage

### Prerequisites

Install and configure Pulumi (with AWS).

### Managing stack

#### Starting / Updating stack

Run `go run . update` to create the Wireguard host.

The script will generate the key pair + wireguard config for the client to connect to the host.

#### Cleaning up

Run `go run . destroy` to clean up the resources.

## Architecture

Go + Pulumi was chosen to allow for programmatic stack creation, e.g. performed by a remote server for a more streamlined process.