package main

func linearSearch(values []int, target int) int {
	for i, v := range values {
		if v == target {
			return i
		}
	}
	return -1
}
