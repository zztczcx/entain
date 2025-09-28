package service

import (
	"testing"
	"time"

	"git.neds.sh/matty/entain/racing/db"
	"git.neds.sh/matty/entain/racing/proto/racing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRacingService_ListRaces_StatusDerivation(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name         string
		repoRaces    []*racing.Race
		repoErr      error
		expectStatus []racing.Race_Status
		expectErr    bool
	}{
		{
			name: "past and future mapped",
			repoRaces: []*racing.Race{
				{Id: 1, AdvertisedStartTime: timestamppb.New(past)},
				{Id: 2, AdvertisedStartTime: timestamppb.New(future)},
			},
			expectStatus: []racing.Race_Status{racing.Race_STATUS_CLOSED, racing.Race_STATUS_OPEN},
		},
		{
			name:         "repo error bubbles",
			repoRaces:    nil,
			repoErr:      context.Canceled,
			expectStatus: nil,
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := db.NewRacesRepoMock(t)
			m.On("List", mock.AnythingOfType("*racing.ListRacesRequestFilter")).Return(tt.repoRaces, tt.repoErr).Once()

			svc := NewRacingService(m)
			got, err := svc.ListRaces(context.Background(), &racing.ListRacesRequest{})
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got.Races, len(tt.expectStatus))
			for i, s := range tt.expectStatus {
				require.Equal(t, s, got.Races[i].Status)
			}
		})
	}
}

func TestRacingService_GetRace_NotFoundAndOK(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	tests := []struct {
		name       string
		repoRace   *racing.Race
		repoErr    error
		expectCode string
	}{
		{
			name:       "not found",
			repoRace:   nil,
			repoErr:    nil,
			expectCode: "NotFound",
		},
		{
			name:       "found",
			repoRace:   &racing.Race{Id: 99, AdvertisedStartTime: timestamppb.New(future)},
			repoErr:    nil,
			expectCode: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := db.NewRacesRepoMock(t)
			if tt.repoRace == nil {
				m.On("Get", int64(99)).Return((*racing.Race)(nil), nil).Once()
			} else {
				m.On("Get", int64(99)).Return(tt.repoRace, nil).Once()
			}

			svc := NewRacingService(m)
			resp, err := svc.GetRace(context.Background(), &racing.GetRaceRequest{Id: 99})

			if tt.expectCode == "NotFound" {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp.Race)
			require.Equal(t, int64(99), resp.Race.Id)
		})
	}
}
