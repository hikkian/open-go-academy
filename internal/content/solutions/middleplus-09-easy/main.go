package main

func PartitionRoute(day int) string {
	if day < 1 || day > 31 {
		return "invalid"
	}
	if day <= 10 {
		return "p1"
	}
	if day <= 20 {
		return "p2"
	}
	return "p3"
}
