package main

func SnapshotRead(v []int) []int {
	if len(v) < 1 {
		return []int{0}
	}
	return []int{v[0]}
}
