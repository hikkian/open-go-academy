package main

func HandoffReady(timeline, currentImpact, nextAction bool) bool {
	return timeline && currentImpact && nextAction
}
