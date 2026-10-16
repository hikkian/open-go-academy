package main

func CapacityClass(workers int, arrival int) string {
	if workers <= 0 || arrival < 0 {
		return "invalid"
	}
	if arrival > workers {
		return "over"
	}
	return "within"
}
