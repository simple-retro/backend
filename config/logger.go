package config

import "go.uber.org/zap"

// NewLogger creates a new zap.Logger based on the provided configuration.
func NewLogger(config *Config) (*zap.Logger, error) {
	if config.Development {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
