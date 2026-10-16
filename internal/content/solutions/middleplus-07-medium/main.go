package main

func ValidSelectivity(matches, total int) bool { return total > 0 && matches >= 0 && matches <= total }
