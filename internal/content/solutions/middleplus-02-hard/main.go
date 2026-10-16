package main

func BenchmarkComparison(baseNs, candidateNs, baseAllocs, candidateAllocs int64) string {
	if baseNs <= 0 || candidateNs <= 0 || baseAllocs < 0 || candidateAllocs < 0 {
		return "invalid"
	}
	if candidateNs < baseNs && candidateAllocs <= baseAllocs {
		return "faster_no_allocation_regression"
	}
	if candidateNs >= baseNs && candidateAllocs >= baseAllocs {
		return "no_metric_improved"
	}
	return "tradeoff"
}
