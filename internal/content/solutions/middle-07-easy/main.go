package main

func RetryAllowed(v []int) []int {
	if len(v) != 2 || v[0] < 1 || v[1] < 1 {
		return []int{0}
	}
	if v[0] < v[1] {
		return []int{1}
	}
	return []int{0}
}
