package main

func VersionMatch(v []int) []int {
	if len(v) != 2 {
		return []int{0}
	}
	if v[0] == v[1] {
		return []int{1}
	}
	return []int{0}
}
