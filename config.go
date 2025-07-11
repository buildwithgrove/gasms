package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Config struct {
		Thresholds Thresholds         `yaml:"thresholds"`
		Networks   map[string]Network `yaml:"networks"`
	} `yaml:"config"`
}

type Thresholds struct {
	WarningThreshold int64 `yaml:"warning_threshold"`
	DangerThreshold  int64 `yaml:"danger_threshold"`
}

type Network struct {
	RPCEndpoint  string   `yaml:"rpc_endpoint"`
	Gateways     []string `yaml:"gateways"`
	Applications []string `yaml:"applications"`
	Bank         string   `yaml:"bank"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
