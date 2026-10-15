package main

func countMatches(values []int, target int) int {
	n := 0
	for _, v := range values {
		if v == target {
			n++
		}
	}
	return n
}
