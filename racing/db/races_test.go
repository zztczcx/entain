package db

import (
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"git.neds.sh/matty/entain/racing/proto/racing"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func Test_applyFilter_QueryOnly(t *testing.T) {
	base := getRaceQueries()[racesList]

	tests := []struct {
		name      string
		filter    *racing.ListRacesRequestFilter
		expectSQL string
	}{
		{
			name:      "nil filter returns base query",
			filter:    nil,
			expectSQL: base,
		},
		{
			name:      "meeting_ids single",
			filter:    &racing.ListRacesRequestFilter{MeetingIds: []int64{1}},
			expectSQL: base + " WHERE meeting_id IN (?)",
		},
		{
			name:      "meeting_ids multiple",
			filter:    &racing.ListRacesRequestFilter{MeetingIds: []int64{1, 2, 3}},
			expectSQL: base + " WHERE meeting_id IN (?,?,?)",
		},
		{
			name:      "show_hidden unset includes hidden (no visible filter)",
			filter:    &racing.ListRacesRequestFilter{},
			expectSQL: base,
		},
		{
			name:      "show_hidden false adds visible=1",
			filter:    &racing.ListRacesRequestFilter{ShowHidden: boolPtr(false)},
			expectSQL: base + " WHERE visible = 1",
		},
		{
			name:      "meeting_ids + show_hidden false",
			filter:    &racing.ListRacesRequestFilter{MeetingIds: []int64{1, 2}, ShowHidden: boolPtr(false)},
			expectSQL: base + " WHERE meeting_id IN (?,?) AND visible = 1",
		},
	}

	r := &racesRepo{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, _ := r.applyFilter(base, tt.filter)
			require.Equal(t, tt.expectSQL, gotSQL)
		})
	}
}

func TestRacesRepo_List_WithSQLMock(t *testing.T) {
	base := getRaceQueries()[racesList]
	cols := []string{"id", "meeting_id", "name", "number", "visible", "advertised_start_time"}

	tests := []struct {
		name      string
		filter    *racing.ListRacesRequestFilter
		expectSQL string
		args      []any
		rows      [][]any
		wantCount int
	}{
		{
			name:      "meeting_ids + show_hidden=false",
			filter:    &racing.ListRacesRequestFilter{MeetingIds: []int64{1, 2}, ShowHidden: boolPtr(false)},
			expectSQL: base + " WHERE meeting_id IN (?,?) AND visible = 1",
			args:      []any{int64(1), int64(2)},
			rows: [][]any{
				{int64(10), int64(1), "Race A", int64(3), true, time.Now()},
				{int64(11), int64(2), "Race B", int64(4), true, time.Now()},
			},
			wantCount: 2,
		},
		{
			name:      "show_hidden unset includes hidden",
			filter:    &racing.ListRacesRequestFilter{},
			expectSQL: base,
			args:      nil,
			rows: [][]any{
				{int64(20), int64(5), "Race C", int64(1), false, time.Now()},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()

			repo := &racesRepo{db: sqlDB}

			r := sqlmock.NewRows(cols)
			for _, row := range tt.rows {
				var drvVals []driver.Value
				for _, v := range row {
					drvVals = append(drvVals, v)
				}
				r = r.AddRow(drvVals...)
			}

			exp := mock.ExpectQuery(regexp.QuoteMeta(tt.expectSQL))
			if tt.args != nil {
				var argVals []driver.Value
				for _, a := range tt.args {
					argVals = append(argVals, a)
				}
				exp = exp.WithArgs(argVals...)
			}
			exp.WillReturnRows(r)

			got, err := repo.List(tt.filter)
			require.NoError(t, err)
			require.Len(t, got, tt.wantCount)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
