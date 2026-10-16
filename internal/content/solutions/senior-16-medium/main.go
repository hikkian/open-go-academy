package main

func CapacityHeadroom(capacity, demand int) string {
	if capacity <= 0 || demand < 0 {
		return "invalid"
	}
	if demand > capacity {
		return "over_limit"
	}
	if demand >= capacity-capacity/5 {
		return "low_headroom"
	}
	return "headroom"
}
