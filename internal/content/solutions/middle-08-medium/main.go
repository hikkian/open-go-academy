package main

func MonotonicSnapshots(v []int) []int {
	last := -1
	for _, x := range v {
		if x < 0 || x < last {
			return []int{0}
		}
		last = x
	}
	return []int{1}
}
