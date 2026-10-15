package main

import "fmt"

func countTo(n int) string {
	if n <= 0 {
		return ""
	}
	if n > 20 {
		n = 20
	}
	result := ""
	for i := 1; i <= n; i++ {
		if i > 1 {
			result += " "
		}
		result += fmt.Sprintf("%d", i)
	}
	return result
}
