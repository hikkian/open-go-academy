package main

func ReleaseEvidence(baselineRuns int, candidateRuns int, faultTested bool) bool {
	return baselineRuns >= 3 && candidateRuns >= 3 && faultTested
}
