package main

func OrderedResults(v []int) []int {
	n := len(v) / 2
	o := make([]int, n)
	seen := make([]bool, n)
	for i := 0; i+1 < len(v); i += 2 {
		k, x := v[i], v[i+1]
		if k >= 0 && k < n && !seen[k] {
			o[k] = x
			seen[k] = true
		}
	}
	r := []int{}
	for i, x := range o {
		if seen[i] {
			r = append(r, x)
		}
	}
	return r
}
