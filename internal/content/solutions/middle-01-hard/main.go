package main

func WorkerLoads(v []int) []int {
	if len(v) != 2 || v[0] <= 0 || v[1] < 0 {
		return []int{}
	}
	o := make([]int, v[0])
	for i := 0; i < v[1]; i++ {
		o[i%v[0]]++
	}
	return o
}
