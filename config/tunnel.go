package config

import (
	"fmt"
	"net"
	"strings"

	"gopkg.in/yaml.v3"
)

type Tunnel struct {
	Network []string `yaml:"network" json:"network"`
	Address string   `yaml:"address" json:"address"`
	Target  string   `yaml:"target" json:"target"`
	Proxy   string   `yaml:"proxy" json:"proxy"`
}

func (t *Tunnel) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		return t.unmarshalString(value.Value)
	}

	type tunnel Tunnel
	var inner tunnel
	if err := value.Decode(&inner); err != nil {
		return err
	}
	*t = Tunnel(inner)
	return nil
}

func (t Tunnel) MarshalYAML() (any, error) {
	type tunnel Tunnel
	return tunnel(t), nil
}

func (t *Tunnel) unmarshalString(value string) error {
	parts := strings.Split(value, ",")
	if len(parts) != 3 && len(parts) != 4 {
		return fmt.Errorf("invalid tunnel config %s", value)
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	network := strings.Split(parts[0], "/")
	for _, item := range network {
		switch item {
		case "tcp", "udp":
		default:
			return fmt.Errorf("invalid tunnel network %s", item)
		}
	}

	for _, address := range []string{parts[1], parts[2]} {
		if _, _, err := net.SplitHostPort(address); err != nil {
			return fmt.Errorf("invalid tunnel target or address %s", address)
		}
	}

	*t = Tunnel{
		Network: network,
		Address: parts[1],
		Target:  parts[2],
	}
	if len(parts) == 4 {
		t.Proxy = parts[3]
	}
	return nil
}
