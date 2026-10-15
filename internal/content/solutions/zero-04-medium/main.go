package main

import "errors"

func sumTo(n int) (int, error) {
	if n < 0 || n > 1_000_000 {
		return 0, errors.New("n вне диапазона")
	}
	total := 0
	for i := 1; i <= n; i++ {
		total += i
	}
	return total, nil
}
