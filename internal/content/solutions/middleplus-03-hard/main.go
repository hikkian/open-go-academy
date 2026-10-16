package main

func RelativeSpreadPercent(values []int) int {
	if len(values) == 0 {
		return -1
	}
	lo, hi := values[0], values[0]
	for _, v := range values {
		if v < 0 {
			return -1
		}
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	if lo == 0 {
		if hi == 0 {
			return 0
		}
		return -1
	}
	delta := hi - lo
	maxInt := int(^uint(0) >> 1)
	if delta > maxInt/100 {
		return -1
	}
	return delta * 100 / lo
}
