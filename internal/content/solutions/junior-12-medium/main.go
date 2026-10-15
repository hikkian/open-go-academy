package main

import (
	"errors"
)

func ApplyDiscount[T interface{ ~int | ~int64 }](price, percent T) (T, error) {
	var z T
	if price < 0 || percent < 0 || percent > 100 {
		return z, errors.New("invalid range")
	}
	return price * (100 - percent) / 100, nil
}
