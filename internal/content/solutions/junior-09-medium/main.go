package main

func binarySearch(sorted []int, target int) int {
	lo, hi := 0, len(sorted)
	for lo < hi {
		m := lo + (hi-lo)/2
		if sorted[m] < target {
			lo = m + 1
		} else {
			hi = m
		}
	}
	if lo < len(sorted) && sorted[lo] == target {
		return lo
	}
	return -1
}
