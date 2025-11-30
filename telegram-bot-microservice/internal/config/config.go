package config

import (
	"log"

	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Service  ServiceConfig  `yaml:"service"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type ServiceConfig struct {
	Port int `yaml:"port"`
}

func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("could not open config file: %v", err)
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		log.Fatalf("could not decode config file: %v", err)
		return nil, err
	}

	return config, nil
}
