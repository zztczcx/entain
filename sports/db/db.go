package db

import (
	"time"

	"syreclabs.com/go/faker"
)

func (r *eventsRepo) seed() error {
	statement, err := r.db.Prepare(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY,
			sport_id INTEGER,
			name TEXT,
			venue TEXT,
			visible INTEGER,
			advertised_start_time DATETIME,
			home_team TEXT,
			away_team TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = statement.Exec()
	if err != nil {
		return err
	}

	// Check if we already have data
	var count int
	err = r.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Already seeded
	}

	sports := []struct {
		id   int64
		name string
	}{
		{1, "Football"},
		{2, "Basketball"},
		{3, "Tennis"},
		{4, "Soccer"},
		{5, "Baseball"},
	}

	venues := []string{
		"Madison Square Garden",
		"Wembley Stadium",
		"Old Trafford",
		"Staples Center",
		"Yankee Stadium",
		"Centre Court Wimbledon",
		"Camp Nou",
		"Emirates Stadium",
	}

	teams := map[int64][]string{
		1: {"Patriots", "Cowboys", "Packers", "Steelers", "49ers", "Giants", "Eagles", "Chiefs"},
		2: {"Lakers", "Celtics", "Warriors", "Bulls", "Heat", "Knicks", "Nets", "Spurs"},
		3: {"Djokovic", "Nadal", "Federer", "Murray", "Tsitsipas", "Medvedev", "Zverev", "Thiem"},
		4: {"Manchester United", "Liverpool", "Arsenal", "Chelsea", "Barcelona", "Real Madrid", "Bayern Munich", "PSG"},
		5: {"Yankees", "Red Sox", "Dodgers", "Giants", "Cubs", "Cardinals", "Astros", "Braves"},
	}

	for i := 1; i <= 100; i++ {
		sport := sports[faker.RandomInt(0, len(sports)-1)]
		venue := venues[faker.RandomInt(0, len(venues)-1)]
		sportTeams := teams[sport.id]

		homeTeam := sportTeams[faker.RandomInt(0, len(sportTeams)-1)]
		awayTeam := sportTeams[faker.RandomInt(0, len(sportTeams)-1)]

		// Ensure home and away teams are different
		for homeTeam == awayTeam {
			awayTeam = sportTeams[faker.RandomInt(0, len(sportTeams)-1)]
		}

		var eventName string
		if sport.name == "Tennis" {
			eventName = homeTeam + " vs " + awayTeam
		} else {
			eventName = homeTeam + " vs " + awayTeam
		}

		// Generate events with times ranging from 1 hour ago to 24 hours in the future
		startTime := time.Now().Add(time.Duration(faker.RandomInt(-60, 1440)) * time.Minute)

		if _, err := r.db.Exec(`
			INSERT INTO events(
				id, 
				sport_id, 
				name, 
				venue, 
				visible, 
				advertised_start_time,
				home_team,
				away_team
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			i,
			sport.id,
			eventName,
			venue,
			faker.RandomInt(0, 1),
			startTime.Format(time.RFC3339),
			homeTeam,
			awayTeam,
		); err != nil {
			return err
		}
	}

	return nil
}
