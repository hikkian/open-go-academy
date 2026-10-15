package main

func sumEven(limit int) int {
	total := 0
	for n := 0; n <= limit; n++ {
		if n%2 == 0 {
			total += n
		}
	}
	return total
}
