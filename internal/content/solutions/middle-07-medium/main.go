package main

func CompareAndSwapModel(v []int) []int {
	if len(v) != 3 {
		return []int{0, 0}
	}
	if v[0] == v[1] {
		return []int{v[2], 1}
	}
	return []int{v[0], 0}
}
