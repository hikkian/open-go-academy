package main

func ScaleLabDecision(baselineRuns int, candidateRuns int) string {
	if baselineRuns <= 0 || candidateRuns <= 0 { return "incomplete" }; return "compare"
}
