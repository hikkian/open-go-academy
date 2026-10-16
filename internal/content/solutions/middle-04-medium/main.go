package main

func BufferLatest(v []int) []int {
	if len(v) == 0 || v[0] <= 0 {
		return []int{}
	}
	n := v[0]
	a := v[1:]
	if len(a) > n {
		a = a[len(a)-n:]
	}
	return append([]int{}, a...)
}
