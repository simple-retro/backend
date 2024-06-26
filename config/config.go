package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Name        string `yaml:"name"`
	Development bool   `yaml:"development"`
	Database    Database
	Server      Server
	Schedule    Schedule
}

type Schedule struct {
	CleanUpDays     int `yaml:"clean_up_days"`
	IntervalMinutes int `yaml:"interval_minutes"`
}

type Server struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	WithCors bool   `yaml:"with_cors"`
}

type Database struct {
	Type    string `yaml:"type"`
	Address string `yaml:"address"`
	Cache   string `yaml:"cache"`
	MaxConn int    `yaml:"max_conn"`
	Schema  string `yaml:"schema"`
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
