package main

func ScaleLabGate(baselineRuns int, candidateRuns int, faultTested bool) string {
	if baselineRuns < 3 || candidateRuns < 3 { return "repeat" }; if !faultTested { return "fault_test" }; return "review"
}
