package main

func clampInt[T interface{ ~int | ~int64 }](value, low, high T) T {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
