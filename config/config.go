package config

import (
	"fmt"
	"os"

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
