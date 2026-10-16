package main

func ConditionalUpdate(v []int) []int {
	if len(v) != 4 || v[1] < 0 || v[1] == int(^uint(0)>>1) {
		return []int{0, 0, 0}
	}
	if v[1] != v[2] {
		return []int{v[0], v[1], 0}
	}
	return []int{v[3], v[1] + 1, 1}
}
