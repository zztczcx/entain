package db

import (
	"testing"

	"git.neds.sh/matty/entain/sports/proto/sports"
	"github.com/stretchr/testify/assert"
)

func TestEventsRepo_applyFilter(t *testing.T) {
	repo := &eventsRepo{}
	baseQuery := "SELECT * FROM events"

	// Helper function to create a pointer to bool
	boolPtr := func(b bool) *bool {
		return &b
	}

	tests := []struct {
		name          string
		filter        *sports.ListEventsRequestFilter
		expectedQuery string
		expectedArgs  []any
	}{
		{
			name:          "nil filter",
			filter:        nil,
			expectedQuery: baseQuery,
			expectedArgs:  nil,
		},
		{
			name:          "empty filter",
			filter:        &sports.ListEventsRequestFilter{},
			expectedQuery: baseQuery + " ORDER BY advertised_start_time ASC",
			expectedArgs:  nil,
		},
		{
			name: "single sport_id",
			filter: &sports.ListEventsRequestFilter{
				SportIds: []int64{1},
			},
			expectedQuery: baseQuery + " WHERE sport_id IN (?) ORDER BY advertised_start_time ASC",
			expectedArgs:  []any{int64(1)},
		},
		{
			name: "multiple sport_ids",
			filter: &sports.ListEventsRequestFilter{
				SportIds: []int64{1, 2, 3},
			},
			expectedQuery: baseQuery + " WHERE sport_id IN (?,?,?) ORDER BY advertised_start_time ASC",
			expectedArgs:  []any{int64(1), int64(2), int64(3)},
		},
		{
			name: "show_hidden false",
			filter: &sports.ListEventsRequestFilter{
				ShowHidden: boolPtr(false),
			},
			expectedQuery: baseQuery + " WHERE visible = 1 ORDER BY advertised_start_time ASC",
			expectedArgs:  []any{},
		},
		{
			name: "show_hidden true",
			filter: &sports.ListEventsRequestFilter{
				ShowHidden: boolPtr(true),
			},
			expectedQuery: baseQuery + " ORDER BY advertised_start_time ASC",
			expectedArgs:  []any{},
		},
		{
			name: "combined filters",
			filter: &sports.ListEventsRequestFilter{
				SportIds:   []int64{1, 2},
				ShowHidden: boolPtr(false),
			},
			expectedQuery: baseQuery + " WHERE sport_id IN (?,?) AND visible = 1 ORDER BY advertised_start_time ASC",
			expectedArgs:  []any{int64(1), int64(2)},
		},
		{
			name: "order by id desc",
			filter: &sports.ListEventsRequestFilter{
				OrderBy: "id desc",
			},
			expectedQuery: baseQuery + " ORDER BY id DESC",
			expectedArgs:  []any{},
		},
		{
			name: "order by name with spaces",
			filter: &sports.ListEventsRequestFilter{
				OrderBy: "  name   asc  ",
			},
			expectedQuery: baseQuery + " ORDER BY name ASC",
			expectedArgs:  []any{},
		},
		{
			name: "invalid order by field",
			filter: &sports.ListEventsRequestFilter{
				OrderBy: "invalid_field",
			},
			expectedQuery: baseQuery,
			expectedArgs:  []any{},
		},
		{
			name: "case insensitive order by",
			filter: &sports.ListEventsRequestFilter{
				OrderBy: "ID DESC",
			},
			expectedQuery: baseQuery + " ORDER BY id DESC",
			expectedArgs:  []any{},
		},
		{
			name: "all options combined",
			filter: &sports.ListEventsRequestFilter{
				SportIds:   []int64{1, 3},
				ShowHidden: boolPtr(false),
				OrderBy:    "name desc",
			},
			expectedQuery: baseQuery + " WHERE sport_id IN (?,?) AND visible = 1 ORDER BY name DESC",
			expectedArgs:  []any{int64(1), int64(3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, actualArgs := repo.applyFilter(baseQuery, tt.filter)

			assert.Equal(t, tt.expectedQuery, actualQuery, "Query should match expected")

			// Handle nil vs empty slice comparison
			if len(tt.expectedArgs) == 0 {
				assert.True(t, len(actualArgs) == 0,
					"Args should be empty, got: %v", actualArgs)
			} else {
				assert.Equal(t, tt.expectedArgs, actualArgs, "Args should match expected")
			}

			// Additional validation for args length when sport_ids are provided
			if tt.filter != nil && len(tt.filter.SportIds) > 0 {
				assert.Len(t, actualArgs, len(tt.filter.SportIds), "Args length should match SportIds length")
			}
		})
	}
}
