package main

func BufferAcceptCount(v []int) []int {
	if len(v) != 2 || v[0] < 0 || v[1] < 0 {
		return []int{0}
	}
	n := v[0]
	if v[1] < n {
		n = v[1]
	}
	return []int{n}
}
