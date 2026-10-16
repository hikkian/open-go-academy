package main

func CapacityBudget(workers int, reservePercent int) int {
	if workers <= 0 || reservePercent < 0 || reservePercent > 100 {
		return -1
	}
	p := 100 - reservePercent
	return workers/100*p + (workers%100)*p/100
}
