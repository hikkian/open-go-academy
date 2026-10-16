package main

func TransferSequence(v []int) []int {
	if len(v) < 2 || v[0] < 0 || v[1] < 0 {
		return []int{}
	}
	a, b := v[0], v[1]
	max := int(^uint(0) >> 1)
	o := []int{}
	for _, x := range v[2:] {
		if x >= 0 && x <= a && x <= max-b {
			a -= x
			b += x
		}
		o = append(o, a, b)
	}
	return o
}
