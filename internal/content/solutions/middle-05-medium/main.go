package main

func RemainingDeadline(v []int) []int {
	if len(v) != 3 || v[0] < 0 || v[1] < 0 || v[2] < 0 {
		return []int{0}
	}
	n := v[0]
	if v[1] < n {
		n = v[1]
	}
	if v[2] >= n {
		return []int{0}
	}
	return []int{n - v[2]}
}
