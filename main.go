package main

import (
	"api/config"
	"api/internal/server"
	"log"
)

func main() {
	config, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("error loading config: %s", err.Error())
	}

	log.Printf("initing service: %s", config.Name)
	server.Start()
}
