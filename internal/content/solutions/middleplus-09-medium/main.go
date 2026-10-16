package main

func ValidPartitionRange(startDay, endDay int) bool {
	return startDay >= 1 && endDay <= 31 && startDay <= endDay
}
