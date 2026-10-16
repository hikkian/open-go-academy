package main

func BatchCount(v []int) []int {
	if len(v) != 2 || v[0] <= 0 || v[1] <= 0 {
		return []int{0}
	}
	return []int{(v[0]-1)/v[1] + 1}
}
