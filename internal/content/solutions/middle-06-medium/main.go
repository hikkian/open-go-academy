package main

func ApplyLockedDeltas(v []int) []int {
	if len(v) < 2 || v[0] < 0 || v[1] < v[0] {
		return []int{}
	}
	b := v[0]
	o := []int{}
	max := int(^uint(0) >> 1)
	min := -max - 1
	for _, d := range v[2:] {
		if (d > 0 && b > max-d) || (d < 0 && b < min-d) {
			break
		}
		n := b + d
		if n < 0 || n > v[1] {
			break
		}
		b = n
		o = append(o, b)
	}
	return o
}
