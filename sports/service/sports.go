package service

import (
	"time"

	"git.neds.sh/matty/entain/sports/db"
	"git.neds.sh/matty/entain/sports/proto/sports"
	"golang.org/x/net/context"
)

type Sports interface {
	// ListEvents will return a collection of sports events.
	ListEvents(ctx context.Context, in *sports.ListEventsRequest) (*sports.ListEventsResponse, error)
}

// sportsService implements the Sports interface.
type sportsService struct {
	eventsRepo db.EventsRepo
}

// NewSportsService instantiates and returns a new sportsService.
func NewSportsService(eventsRepo db.EventsRepo) *sportsService {
	return &sportsService{eventsRepo: eventsRepo}
}

func (s *sportsService) ListEvents(ctx context.Context, in *sports.ListEventsRequest) (*sports.ListEventsResponse, error) {
	events, err := s.eventsRepo.List(in.Filter)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for _, e := range events {
		if e.AdvertisedStartTime.AsTime().After(now) {
			e.Status = sports.Event_STATUS_OPEN
		} else {
			e.Status = sports.Event_STATUS_CLOSED
		}
	}

	return &sports.ListEventsResponse{Events: events}, nil
}
