package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Config struct {
		Networks map[string]Network `yaml:"networks"`
	} `yaml:"config"`
}

type Network struct {
	RPCEndpoint  string   `yaml:"rpc_endpoint"`
	Gateways     []string `yaml:"gateways"`
	Applications []string `yaml:"applications"`
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
