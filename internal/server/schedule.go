package server

import (
	"api/config"
	"context"
	"log"
	"time"
)

func (c *controller) StartSchedule() {
	config := config.Get()
	go func() {
		ticker := time.NewTicker(time.Duration(config.Schedule.IntervalMinutes) * time.Minute)

		for {
			select {
			case <-ticker.C:
				c.CleanUp()
			}
		}
	}()
}

func (c *controller) CleanUp() {
	log.Println("starting clean up routine")
	ctx := context.Background()
	if err := c.service.CleanUpRetros(ctx); err != nil {
		log.Printf("error running clean up routine: %s", err.Error())
	}
}
