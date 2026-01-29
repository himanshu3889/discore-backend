package gateway

import (
	"os"

	"gopkg.in/yaml.v3"
)

// RouteConfig holds your route rules
type RouteConfig struct {
	Routes []struct {
		Path    string `yaml:"path"`
		Auth    bool   `yaml:"auth"`
		Module  bool   `yaml:"module"`
		Address string `yaml:"address"`
	} `yaml:"gateway"`
}

// LoadConfig reads the YAML file
func LoadGatewayConfig(filename string) (*RouteConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config RouteConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
