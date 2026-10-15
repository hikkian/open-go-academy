package main

import (
	"errors"
	"strconv"
)

func parseLimit(s string) (int, error) {
	n, e := strconv.Atoi(s)
	if e != nil || n <= 0 {
		return 0, errors.New("invalid positive integer")
	}
	return n, nil
}
