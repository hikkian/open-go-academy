package main

func Median(values []int) int {
	if len(values) == 0 {
		return 0
	}
	v := append([]int(nil), values...)
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
	n := len(v)
	if n%2 == 1 {
		return v[n/2]
	}
	a, b := v[n/2-1], v[n/2]
	if (a < 0) != (b < 0) {
		return (a + b) / 2
	}
	return a/2 + b/2 + (a%2+b%2)/2
}
