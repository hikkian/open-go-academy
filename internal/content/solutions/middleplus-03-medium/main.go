package main

func ValidBenchmarkSample(values []int) bool {
	if len(values) < 3 {
		return false
	}
	for _, v := range values {
		if v <= 0 {
			return false
		}
	}
	return true
}
