package db

import (
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"git.neds.sh/matty/entain/racing/proto/racing"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
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
			expectSQL: base + " WHERE meeting_id IN (?) ORDER BY advertised_start_time ASC",
		},
		{
			name:      "meeting_ids multiple",
			filter:    &racing.ListRacesRequestFilter{MeetingIds: []int64{1, 2, 3}},
			expectSQL: base + " WHERE meeting_id IN (?,?,?) ORDER BY advertised_start_time ASC",
		},
		{
			name:      "show_hidden unset includes hidden (no visible filter)",
			filter:    &racing.ListRacesRequestFilter{},
			expectSQL: base + " ORDER BY advertised_start_time ASC",
		},
		{
			name:      "show_hidden false adds visible=1",
			filter:    &racing.ListRacesRequestFilter{ShowHidden: boolPtr(false)},
			expectSQL: base + " WHERE visible = 1 ORDER BY advertised_start_time ASC",
		},
		{
			name:      "meeting_ids + show_hidden false",
			filter:    &racing.ListRacesRequestFilter{MeetingIds: []int64{1, 2}, ShowHidden: boolPtr(false)},
			expectSQL: base + " WHERE meeting_id IN (?,?) AND visible = 1 ORDER BY advertised_start_time ASC",
		},
		{
			name:      "order by name",
			filter:    &racing.ListRacesRequestFilter{OrderBy: "name"},
			expectSQL: base + " ORDER BY name ASC",
		},
		{
			name:      "order by default",
			filter:    &racing.ListRacesRequestFilter{},
			expectSQL: base + " ORDER BY advertised_start_time ASC",
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
			expectSQL: base + " ORDER BY advertised_start_time ASC",
			args:      nil,
			rows: [][]any{
				{int64(20), int64(5), "Race C", int64(1), false, time.Now()},
			},
			wantCount: 1,
		},
		{
			name:      "explicit desc",
			filter:    &racing.ListRacesRequestFilter{OrderBy: "advertised_start_time desc"},
			expectSQL: base + " ORDER BY advertised_start_time DESC",
			args:      nil,
			rows:      [][]any{},
			wantCount: 0,
		},
		{
			name:      "order by name asc",
			filter:    &racing.ListRacesRequestFilter{OrderBy: "name"},
			expectSQL: base + " ORDER BY name ASC",
			args:      nil,
			rows:      [][]any{},
			wantCount: 0,
		},
		{
			name:      "order by number desc",
			filter:    &racing.ListRacesRequestFilter{OrderBy: "number desc"},
			expectSQL: base + " ORDER BY number DESC",
			args:      nil,
			rows:      [][]any{},
			wantCount: 0,
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

func TestRacesRepo_Get_WithSQLMock(t *testing.T) {
	baseGet := getRaceQueries()[racesGet]
	cols := []string{"id", "meeting_id", "name", "number", "visible", "advertised_start_time"}

	tests := []struct {
		name      string
		id        int64
		expectSQL string
		row       []any
		willErr   error
		wantNil   bool
	}{
		{
			name:      "found",
			id:        42,
			expectSQL: baseGet,
			row:       []any{int64(42), int64(5), "Found Race", int64(3), true, time.Now()},
			willErr:   nil,
			wantNil:   false,
		},
		{
			name:      "not found (no rows)",
			id:        7,
			expectSQL: baseGet,
			row:       nil, // no rows
			willErr:   nil,
			wantNil:   true,
		},
		{
			name:      "db error",
			id:        8,
			expectSQL: baseGet,
			row:       nil,
			willErr:   assert.AnError,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()

			repo := &racesRepo{db: sqlDB}

			exp := mock.ExpectQuery(regexp.QuoteMeta(tt.expectSQL)).WithArgs(tt.id)
			if tt.willErr != nil {
				exp.WillReturnError(tt.willErr)
			} else {
				if tt.row == nil {
					exp.WillReturnRows(sqlmock.NewRows(cols))
				} else {
					var drvVals []driver.Value
					for _, v := range tt.row {
						drvVals = append(drvVals, v)
					}
					exp.WillReturnRows(sqlmock.NewRows(cols).AddRow(drvVals...))
				}
			}

			got, err := repo.Get(tt.id)
			if tt.willErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.wantNil {
				require.Nil(t, got)
			} else {
				require.NotNil(t, got)
				require.Equal(t, tt.id, got.Id)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
