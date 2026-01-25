package schedule

import (
	"api/config"
	"api/internal/service"
	"context"
	"log"
	"time"

	"go.uber.org/fx"
)

type Schedule struct {
	service *service.Service
	config  *config.Config
	stopCh  chan struct{}
}

type ScheduleParams struct {
	fx.In
	Service   *service.Service
	Config    *config.Config
	Lifecycle fx.Lifecycle
}

func New(p ScheduleParams) *Schedule {
	s := &Schedule{
		service: p.Service,
		config:  p.Config,
		stopCh:  make(chan struct{}),
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			s.start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.stop()
			return nil
		},
	})

	return s
}

func (s *Schedule) start() {
	go func() {
		ticker := time.NewTicker(time.Duration(s.config.Schedule.IntervalMinutes) * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanUp()
			case <-s.stopCh:
				log.Println("stopping schedule")
				return
			}
		}
	}()
}

func (s *Schedule) stop() {
	close(s.stopCh)
}

func (s *Schedule) cleanUp() {
	log.Println("starting clean up routine")
	ctx := context.Background()
	if err := s.service.CleanUpRetros(ctx); err != nil {
		log.Printf("error running clean up routine: %s", err.Error())
	}
}
