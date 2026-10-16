package main

func ReadCommittedReads(v []int) []int {
	o := []int{}
	for i := 0; i+1 < len(v); i += 2 {
		o = append(o, v[i+1])
	}
	return o
}
