package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type TunnelConfig struct {
	Token  string `yaml:"token"`
	Server string `yaml:"server"`
	RDP    string `yaml:"rdp_port"`
	WinRM  string `yaml:"winrm_port"`
}

func DefaultConfig() *TunnelConfig {
	return &TunnelConfig{
		RDP:   "127.0.0.1:3389",
		WinRM: "127.0.0.1:5986",
	}
}

func LoadConfig(path string) (*TunnelConfig, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required in config")
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("server is required in config")
	}

	return cfg, nil
}
