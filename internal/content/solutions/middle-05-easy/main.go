package main

func MinDeadline(v []int) []int {
	if len(v) != 2 || v[0] <= 0 || v[1] <= 0 {
		return []int{0}
	}
	if v[0] < v[1] {
		return []int{v[0]}
	}
	return []int{v[1]}
}
