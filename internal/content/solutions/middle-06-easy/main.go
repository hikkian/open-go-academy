package main

func BoundedBalance(v []int) []int {
	if len(v) != 3 || v[0] < 0 || v[2] < 0 || v[0] > v[2] {
		return []int{0, 0}
	}
	if v[1] > 0 && v[0] > int(^uint(0)>>1)-v[1] {
		return []int{v[0], 0}
	}
	n := v[0] + v[1]
	if n < 0 || n > v[2] {
		return []int{v[0], 0}
	}
	return []int{n, 1}
}
