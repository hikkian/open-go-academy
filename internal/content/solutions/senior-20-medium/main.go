package main

func CanExpand(errorPercent, threshold, sampleCount int) bool {
	return errorPercent >= 0 && threshold >= 0 && threshold <= 100 && errorPercent <= threshold && sampleCount >= 100
}
