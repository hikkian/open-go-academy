package main

func ValidProfileCounts(flat, cumulative, total int) bool {
	return total > 0 && flat >= 0 && cumulative >= flat && cumulative <= total
}
