package db

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"git.neds.sh/matty/entain/racing/proto/racing"
)

// RacesRepo provides repository access to races.
//
//go:generate mockery --name RacesRepo --structname RacesRepoMock --dir . --output . --outpkg db --filename races_repo_mock.go
type RacesRepo interface {
	// Init will initialise our races repository.
	Init() error

	// List will return a list of races.
	List(filter *racing.ListRacesRequestFilter) ([]*racing.Race, error)

	// Get returns a single race by id.
	Get(id int64) (*racing.Race, error)
}

type racesRepo struct {
	db   *sql.DB
	init sync.Once
}

// NewRacesRepo creates a new races repository.
func NewRacesRepo(db *sql.DB) RacesRepo {
	return &racesRepo{db: db}
}

// Init prepares the race repository dummy data.
func (r *racesRepo) Init() error {
	var err error

	r.init.Do(func() {
		// For test/example purposes, we seed the DB with some dummy races.
		err = r.seed()
	})

	return err
}

func (r *racesRepo) List(filter *racing.ListRacesRequestFilter) ([]*racing.Race, error) {
	var (
		err   error
		query string
		args  []any
	)

	query = getRaceQueries()[racesList]

	query, args = r.applyFilter(query, filter)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return r.scanRaces(rows)
}

func (r *racesRepo) Get(id int64) (*racing.Race, error) {
	query := getRaceQueries()[racesGet]
	row := r.db.QueryRow(query, id)
	var race racing.Race
	var advertisedStart time.Time
	if err := row.Scan(&race.Id, &race.MeetingId, &race.Name, &race.Number, &race.Visible, &advertisedStart); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	race.AdvertisedStartTime = timestamppb.New(advertisedStart)
	return &race, nil
}

func (r *racesRepo) applyFilter(query string, filter *racing.ListRacesRequestFilter) (string, []any) {
	var (
		clauses []string
		args    []any
	)

	if filter == nil {
		return query, args
	}

	if len(filter.MeetingIds) > 0 {
		clauses = append(clauses, "meeting_id IN ("+strings.Repeat("?,", len(filter.MeetingIds)-1)+"?)")

		for _, meetingID := range filter.MeetingIds {
			args = append(args, meetingID)
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
			case "id", "meeting_id", "name", "number", "visible", "advertised_start_time":
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

func (m *racesRepo) scanRaces(
	rows *sql.Rows,
) ([]*racing.Race, error) {
	var races []*racing.Race

	for rows.Next() {
		var race racing.Race
		var advertisedStart time.Time

		if err := rows.Scan(&race.Id, &race.MeetingId, &race.Name, &race.Number, &race.Visible, &advertisedStart); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}

			return nil, err
		}

		race.AdvertisedStartTime = timestamppb.New(advertisedStart)

		races = append(races, &race)
	}

	return races, nil
}
