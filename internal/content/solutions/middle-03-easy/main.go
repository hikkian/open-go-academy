package main

func TakeChannelPrefix(v []int) []int {
	if len(v) == 0 || v[0] <= 0 {
		return []int{}
	}
	n := v[0]
	if n > len(v)-1 {
		n = len(v) - 1
	}
	return append([]int{}, v[1:1+n]...)
}
