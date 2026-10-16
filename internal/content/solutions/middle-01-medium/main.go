package main

func AssignWorkers(v []int) []int {
	if len(v) != 2 || v[0] <= 0 || v[1] < 0 {
		return []int{}
	}
	o := make([]int, v[1])
	for i := range o {
		o[i] = i % v[0]
	}
	return o
}
