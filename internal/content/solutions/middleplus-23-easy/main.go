package main

func IncidentPriority(impact int, ongoing bool, eventTimes []int, observed []bool) string {
	if impact < 0 || len(eventTimes) == 0 || len(eventTimes) != len(observed) {
		return "invalid"
	}
	for i, t := range eventTimes {
		if t < 0 || (i > 0 && t < eventTimes[i-1]) {
			return "invalid"
		}
	}
	if ongoing {
		return "respond"
	}
	for _, ok := range observed {
		if !ok {
			return "review_gaps"
		}
	}
	return "review"
}
