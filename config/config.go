package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TrustedProxyIPs []string `yaml:"trustedProxyIPs"`
	Public          struct {
		Port int `yaml:"port"`
	} `yaml:"public"`
	Admin struct {
		Port       int      `yaml:"port"`
		AllowedIPs []string `yaml:"allowedIPs"`
	} `yaml:"admin"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseConfig(data)
}

func ParseConfig(data []byte) (*Config, error) {
	cfg := &Config{}
	cfg.Public.Port = 8080
	cfg.Admin.Port = 9000
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Public.Port == cfg.Admin.Port {
		return nil, fmt.Errorf("public and admin port must differ")
	}
	return cfg, nil
}

// ParseNets parses a list of CIDRs or single IPs.
func ParseNets(list []string) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	for _, s := range list {
		s = strings.TrimSpace(s)
		if !strings.Contains(s, "/") {
			ip := net.ParseIP(s)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP %q", s)
			}
			if ip4 := ip.To4(); ip4 != nil {
				s += "/32"
			} else {
				s += "/128"
			}
		}
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", s, err)
		}
		nets = append(nets, n)
	}
	return nets, nil
}

func ipInNets(ip net.IP, nets []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
