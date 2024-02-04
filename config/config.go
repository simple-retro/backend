package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Name     string `yaml:"name"`
	Database Database
	Server   Server
}

type Server struct {
	Port int `yaml:"port"`
}

type Database struct {
	Address string `yaml:"address"`
	MaxConn int    `yaml:"max_conn"`
}

var config *Config

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	config = &conf
	return config, nil
}

func Get() *Config {
	return config
}
