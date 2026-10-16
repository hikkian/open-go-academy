package main

func BufferAccounting(v []int) []int {
	if len(v) != 3 || v[0] < 0 || v[1] < 0 || v[2] < 0 {
		return []int{0, 0}
	}
	a := v[1]
	if a > v[0] {
		a = v[0]
	}
	d := v[1] - a
	c := v[2]
	if c > a {
		c = a
	}
	return []int{a - c, d}
}
