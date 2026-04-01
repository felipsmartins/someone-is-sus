package httpmessage

// CommonGames is a set of common games.
// It makes life easier instead access database for this small task.
var CommonGames = map[string]int64{
	"team_fortress_2": 1,
}

// ReportData represents the request body for player reporting
type ReportData struct {
	ProfileURL string `json:"profile_url"`
	ReportedBy string `json:"reported_by"`
	GameID     int64  `json:"game_id"`
}
