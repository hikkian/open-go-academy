package main

func minMax(values []int) (int, int, bool) {
	if len(values) == 0 {
		return 0, 0, false
	}
	lo, hi := values[0], values[0]
	for _, v := range values[1:] {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	return lo, hi, true
}
