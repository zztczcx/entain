package service_test

import (
	"errors"
	"testing"
	"time"

	"git.neds.sh/matty/entain/sports/db"
	"git.neds.sh/matty/entain/sports/proto/sports"
	"git.neds.sh/matty/entain/sports/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func TestSportsService_ListEvents(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		events  []*sports.Event
		mockErr error
		wantErr bool
		assert  func(t *testing.T, resp *sports.ListEventsResponse)
	}{
		{
			name: "sets status open/closed based on time",
			events: []*sports.Event{
				{Id: 1, SportId: 10, Name: "Future Match", AdvertisedStartTime: timestamppb.New(now.Add(1 * time.Hour))},
				{Id: 2, SportId: 20, Name: "Past Match", AdvertisedStartTime: timestamppb.New(now.Add(-1 * time.Hour))},
			},
			assert: func(t *testing.T, resp *sports.ListEventsResponse) {
				require.Len(t, resp.Events, 2)
				var e1, e2 *sports.Event
				for _, e := range resp.Events {
					switch e.Id {
					case 1:
						e1 = e
					case 2:
						e2 = e
					}
				}
				require.NotNil(t, e1)
				require.NotNil(t, e2)
				require.Equal(t, sports.Event_STATUS_OPEN, e1.Status)
				require.Equal(t, sports.Event_STATUS_CLOSED, e2.Status)
			},
		},
		{
			name:    "propagates repo error",
			mockErr: errors.New("boom"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := db.NewEventsRepoMock(t)
			repo.On("List", mock.Anything).Return(tt.events, tt.mockErr)
			svc := service.NewSportsService(repo)

			resp, err := svc.ListEvents(context.Background(), &sports.ListEventsRequest{Filter: &sports.ListEventsRequestFilter{}})
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tt.assert != nil {
				tt.assert(t, resp)
			}
		})
	}
}
