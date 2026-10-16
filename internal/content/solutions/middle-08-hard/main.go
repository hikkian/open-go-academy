package main

func FindFirstRegression(v []int) []int {
	for i := 1; i < len(v); i++ {
		if v[i] < v[i-1] {
			return []int{i, v[i-1], v[i]}
		}
	}
	return []int{-1}
}
