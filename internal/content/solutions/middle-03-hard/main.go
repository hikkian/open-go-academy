package main

func InterleaveProducers(v []int) []int {
	if len(v) < 2 || v[0] <= 0 || v[1] < 0 {
		return []int{}
	}
	p, r := v[0], v[1]
	o := []int{}
	for j := 0; j < r; j++ {
		for i := 0; i < p; i++ {
			k := 2 + i*r + j
			if k < len(v) {
				o = append(o, v[k])
			}
		}
	}
	return o
}
