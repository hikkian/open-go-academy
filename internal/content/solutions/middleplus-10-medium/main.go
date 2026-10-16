package main

func ValidReplicaLag(lagSeconds, maxLag int) bool {
	return lagSeconds >= 0 && maxLag >= 0 && lagSeconds <= maxLag
}
