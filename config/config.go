package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Name          string `yaml:"name"`
	Development   bool   `yaml:"development"`
	Database      Database
	Server        Server
	Schedule      Schedule
	SessionSecret string
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

func Load(filename string, envFilename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	err = godotenv.Load(envFilename)
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	conf.SessionSecret = os.Getenv("SESSION_SECRET")
	if conf.SessionSecret == "" {
		return nil, fmt.Errorf("missing session secret in environment variables")
	}

	config = &conf
	return config, nil
}

func Get() *Config {
	return config
}

// ConfigPaths holds the paths for config and env files
type ConfigPaths struct {
	ConfigFile string
	EnvFile    string
}

// NewConfig creates a new config instance for FX dependency injection
func NewConfig(paths ConfigPaths) (*Config, error) {
	return Load(paths.ConfigFile, paths.EnvFile)
}
