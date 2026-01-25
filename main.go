package main

import (
	"context"
	"log"

	"api/config"
	"api/internal/repository"
	"api/internal/schedule"
	"api/internal/server"
	"api/internal/service"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.Provide(
			// Config file paths
			func() config.ConfigPaths {
				return config.ConfigPaths{
					ConfigFile: "config/config.yaml",
					EnvFile:    "config/.env",
				}
			},
			config.NewConfig,
			config.NewLogger,
			repository.NewSQLite,
			repository.NewWebSocket,
			service.New,
		),
		fx.Invoke(
			// Load all retrospectives on startup
			func(svc *service.Service) error {
				if err := svc.LoadAllRetrospectives(context.Background()); err != nil {
					log.Printf("error loading retrospectives: %s", err.Error())
					return err
				}
				return nil
			},

			// Ensure logger is synced on shutdown
			func(logger *zap.Logger, lc fx.Lifecycle) {
				lc.Append(fx.Hook{
					OnStop: func(ctx context.Context) error {
						return logger.Sync()
					},
				})
			},

			server.New,
			schedule.New,
		),
	).Run()
}
