package main

import (
	"api/config"
	"api/internal/repository"
	"api/internal/schedule"
	"api/internal/server"
	"api/internal/service"
	"context"
	"log"
)

func main() {
	config, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("error loading config: %s", err.Error())
	}

	repo, err := repository.NewSQLite()
	if err != nil {
		log.Fatalf("error creating repository: %s", err.Error())
	}

	wsrepo, err := repository.NewWebSocket()
	if err != nil {
		log.Fatalf("error creating repository: %s", err.Error())
	}

	service := service.New(repo, wsrepo)
	service.LoadAllRetrospectives(context.Background())

	controller := server.New(service)

	schedule := schedule.New(service)
	schedule.Start()

	log.Printf("initing service: %s", config.Name)
	controller.Start()
}
