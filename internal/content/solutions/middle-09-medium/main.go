package main

func CancelChildren(v []int) []int {
	o := []int{}
	for i := 0; i+1 < len(v); i += 2 {
		if v[i] != 0 && v[i+1] == 0 {
			o = append(o, i/2)
		}
	}
	return o
}
