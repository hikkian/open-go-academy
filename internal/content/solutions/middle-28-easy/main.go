package main

func ConfigStatus(dbURL string) string {
	if dbURL == "" {
		return "missing_database"
	}
	if len(dbURL) < 8 {
		return "invalid_database"
	}
	return "configured"
}
