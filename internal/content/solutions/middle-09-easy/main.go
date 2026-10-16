package main

func BlockedSenders(v []int) []int {
	if len(v) != 3 || v[0] < 0 || v[1] < 0 || v[2] < 0 {
		return []int{0}
	}
	free := v[0] - v[1]
	if free < 0 {
		free = 0
	}
	if v[2] > free {
		return []int{v[2] - free}
	}
	return []int{0}
}
