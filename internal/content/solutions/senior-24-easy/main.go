package main

func ActionPriority(impact, likelihood int) string {
	if impact < 0 || likelihood < 0 {
		return "invalid"
	}
	if impact >= 3 && likelihood >= 3 {
		return "now"
	}
	if impact >= 2 || likelihood >= 2 {
		return "schedule"
	}
	return "track"
}
