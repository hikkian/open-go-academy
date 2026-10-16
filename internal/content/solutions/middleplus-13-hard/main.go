package main

func CapacityDecision(workers int, arrival int, reservePercent int) string {
	if workers <= 0 || arrival < 0 || reservePercent < 0 || reservePercent > 100 {
		return "invalid"
	}
	p := 100 - reservePercent
	budget := workers/100*p + (workers%100)*p/100
	if arrival > budget {
		return "over"
	}
	return "within"
}
