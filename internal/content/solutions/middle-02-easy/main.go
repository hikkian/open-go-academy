package main

func CompletedCount(v []int) []int {
	n := 0
	for _, x := range v {
		if x == 1 {
			n++
		}
	}
	return []int{n}
}
