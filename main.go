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
			server.New,
			schedule.New,
		),
	).Run()
}
