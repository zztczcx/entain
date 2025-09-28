package service

import (
	"time"

	"git.neds.sh/matty/entain/racing/db"
	"git.neds.sh/matty/entain/racing/proto/racing"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Racing interface {
	// ListRaces will return a collection of races.
	ListRaces(ctx context.Context, in *racing.ListRacesRequest) (*racing.ListRacesResponse, error)
	// GetRace returns a single race by ID.
	GetRace(ctx context.Context, in *racing.GetRaceRequest) (*racing.GetRaceResponse, error)
}

// racingService implements the Racing interface.
type racingService struct {
	racesRepo db.RacesRepo
}

// NewRacingService instantiates and returns a new racingService.
func NewRacingService(racesRepo db.RacesRepo) Racing {
	return &racingService{racesRepo}
}

func (s *racingService) ListRaces(ctx context.Context, in *racing.ListRacesRequest) (*racing.ListRacesResponse, error) {
	races, err := s.racesRepo.List(in.Filter)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for _, r := range races {
		if r.AdvertisedStartTime.AsTime().After(now) {
			r.Status = racing.Race_STATUS_OPEN
		} else {
			r.Status = racing.Race_STATUS_CLOSED
		}
	}

	return &racing.ListRacesResponse{Races: races}, nil
}

func (s *racingService) GetRace(ctx context.Context, in *racing.GetRaceRequest) (*racing.GetRaceResponse, error) {
	race, err := s.racesRepo.Get(in.Id)
	if err != nil {
		return nil, err
	}
	if race == nil {
		return nil, status.Error(codes.NotFound, "race not found")
	}

	now := time.Now()
	if race.AdvertisedStartTime.AsTime().After(now) {
		race.Status = racing.Race_STATUS_OPEN
	} else {
		race.Status = racing.Race_STATUS_CLOSED
	}

	return &racing.GetRaceResponse{Race: race}, nil
}
