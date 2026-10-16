package main

func IncidentLevel(impact int, suspected bool) string {
	if impact < 0 {
		return "invalid"
	}
	if suspected {
		return "critical_review"
	}
	if impact >= 3 {
		return "major"
	}
	return "routine"
}
