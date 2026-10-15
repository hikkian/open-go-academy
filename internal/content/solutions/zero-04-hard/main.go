package main

import (
	"errors"
	"fmt"
)

func multiplicationTable(n int) (string, error) {
	if n < 0 || n > 12 {
		return "", errors.New("n вне диапазона")
	}
	result := ""
	for i := 1; i <= 10; i++ {
		if i > 1 {
			result += "\n"
		}
		result += fmt.Sprintf("%d x %d = %d", n, i, n*i)
	}
	return result, nil
}
