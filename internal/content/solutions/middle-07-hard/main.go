package main

func SaturatingAtomicAdd(v []int) []int {
	if len(v) != 3 || v[0] < 0 || v[1] < 0 || v[2] < 0 || v[0] > v[2] {
		return []int{0}
	}
	if v[1] > v[2]-v[0] {
		return []int{v[2]}
	}
	return []int{v[0] + v[1]}
}
