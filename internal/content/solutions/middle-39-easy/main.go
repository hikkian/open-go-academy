package main

func LoadFits(offered, capacity int) bool {
	return offered >= 0 && capacity > 0 && offered <= capacity
}
