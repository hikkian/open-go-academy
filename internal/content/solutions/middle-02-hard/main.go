package main

func FirstCancelledIndex(v []int) []int {
	for i, x := range v {
		if x == -1 {
			return []int{i}
		}
	}
	return []int{-1}
}
