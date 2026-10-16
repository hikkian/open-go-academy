package main

func DeadlineChain(v []int) []int {
	o := make([]int, len(v))
	m := int(^uint(0) >> 1)
	for i, x := range v {
		if x <= 0 {
			x = 0
		}
		if x < m {
			m = x
		}
		o[i] = m
	}
	return o
}
