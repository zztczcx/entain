package db

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"git.neds.sh/matty/entain/sports/proto/sports"
)

// EventsRepo provides repository access to sports events.
//
//go:generate mockery --name EventsRepo --structname EventsRepoMock --dir . --output . --outpkg db --filename events_repo_mock.go
type EventsRepo interface {
	// Init will initialise our events repository.
	Init() error

	// List will return a list of sports events.
	List(filter *sports.ListEventsRequestFilter) ([]*sports.Event, error)
}

type eventsRepo struct {
	db   *sql.DB
	init sync.Once
}

// NewEventsRepo creates a new events repository.
func NewEventsRepo(db *sql.DB) EventsRepo {
	return &eventsRepo{db: db}
}

// Init prepares the event repository dummy data.
func (r *eventsRepo) Init() error {
	var err error

	r.init.Do(func() {
		// For test/example purposes, we seed the DB with some dummy events.
		err = r.seed()
	})

	return err
}

func (r *eventsRepo) List(filter *sports.ListEventsRequestFilter) ([]*sports.Event, error) {
	var (
		err   error
		query string
		args  []any
	)

	query = getEventQueries()[eventsList]

	query, args = r.applyFilter(query, filter)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return r.scanEvents(rows)
}

func (r *eventsRepo) applyFilter(query string, filter *sports.ListEventsRequestFilter) (string, []any) {
	var (
		clauses []string
		args    []any
	)

	if filter == nil {
		return query, args
	}

	if len(filter.SportIds) > 0 {
		clauses = append(clauses, "sport_id IN ("+strings.Repeat("?,", len(filter.SportIds)-1)+"?)")

		for _, sportID := range filter.SportIds {
			args = append(args, sportID)
		}
	}

	// show_hidden semantics: unset or true => include hidden; false => only visible
	if filter.ShowHidden != nil && !*filter.ShowHidden {
		clauses = append(clauses, "visible = 1")
	}

	if len(clauses) != 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	if filter.OrderBy != "" {
		ob := strings.ToLower(strings.TrimSpace(filter.OrderBy))
		tokens := strings.Fields(ob)
		if len(tokens) >= 1 {
			switch tokens[0] {
			case "id", "sport_id", "name", "venue", "visible", "advertised_start_time", "home_team", "away_team":
				orderField := tokens[0]
				orderDir := "ASC"
				if len(tokens) >= 2 {
					switch tokens[1] {
					case "desc":
						orderDir = "DESC"
					case "asc":
						orderDir = "ASC"
					}
				}
				query += " ORDER BY " + orderField + " " + orderDir
			default:
				// unknown column ignored to avoid injection; keep defaults
			}
		}
	} else {
		query += " ORDER BY advertised_start_time ASC"
	}

	return query, args
}

func (m *eventsRepo) scanEvents(
	rows *sql.Rows,
) ([]*sports.Event, error) {
	var events []*sports.Event

	for rows.Next() {
		var event sports.Event
		var advertisedStart time.Time

		if err := rows.Scan(&event.Id, &event.SportId, &event.Name, &event.Venue, &event.Visible, &advertisedStart, &event.HomeTeam, &event.AwayTeam); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}

			return nil, err
		}

		event.AdvertisedStartTime = timestamppb.New(advertisedStart)

		events = append(events, &event)
	}

	return events, nil
}
