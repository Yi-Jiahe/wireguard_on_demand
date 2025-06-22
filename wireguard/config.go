package wireguard

import (
	"fmt"
	"strings"
)

type Config struct {
	Interface Interface
	Peers     []Peer
}

type Interface struct {
	PrivateKey string
	ListenPort int
	Address    string
	DNS        string
}

type Peer struct {
	PublicKey  string
	Endpoint   string
	AllowedIps []string
}

func (c *Config) String() string {
	s := fmt.Sprintf(`%s
`, c.Interface.String())

	for _, p := range c.Peers {
		s += fmt.Sprintf(`%s
`, p.String())
	}

	return s
}

func (i *Interface) String() string {
	s := fmt.Sprintf(`[Interface]
PrivateKey = %s`, i.PrivateKey)

	if i.ListenPort != 0 {
		s += fmt.Sprintf(`
ListenPort = %d`, i.ListenPort)
	}

	if i.Address != "" {
		s += fmt.Sprintf(`
Address = %s`, i.Address)
	}

	if i.DNS != "" {
		s += fmt.Sprintf(`
DNS = %s`, i.DNS)
	}

	return s
}

func (p *Peer) String() string {
	s := fmt.Sprintf(`[Peer]
PublicKey = %s`, p.PublicKey)

	if p.Endpoint != "" {
		s += fmt.Sprintf(`
Endpoint = %s`, p.Endpoint)
	}

	s += fmt.Sprintf(`
AllowedIPs = %s`, strings.Join(p.AllowedIps, ","))
	return s
}
