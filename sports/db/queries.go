package db

const (
	eventsList = "list"
)

func getEventQueries() map[string]string {
	return map[string]string{
		eventsList: `
			SELECT 
				id, 
				sport_id, 
				name, 
				venue, 
				visible, 
				advertised_start_time,
				home_team,
				away_team
			FROM events
		`,
	}
}
