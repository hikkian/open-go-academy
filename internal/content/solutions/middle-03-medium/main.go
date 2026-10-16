package main

func DrainThroughSentinel(v []int) []int {
	if len(v) == 0 {
		return []int{}
	}
	o := []int{}
	for _, x := range v[1:] {
		if x == v[0] {
			break
		}
		o = append(o, x)
	}
	return o
}
