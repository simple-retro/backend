package schedule

import (
	"api/config"
	"api/internal/service"
	"context"
	"log"
	"time"
)

type schedule struct {
	service *service.Service
}

func New(s *service.Service) *schedule {
	return &schedule{
		service: s,
	}
}

func (s *schedule) Start() {
	config := config.Get()
	go func() {
		ticker := time.NewTicker(time.Duration(config.Schedule.IntervalMinutes) * time.Minute)

		for {
			select {
			case <-ticker.C:
				s.cleanUp()
			}
		}
	}()
}

func (s *schedule) cleanUp() {
	log.Println("starting clean up routine")
	ctx := context.Background()
	if err := s.service.CleanUpRetros(ctx); err != nil {
		log.Printf("error running clean up routine: %s", err.Error())
	}
}
