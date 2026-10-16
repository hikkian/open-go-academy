package main

func OptimizationEvidence(beforeFlat, afterFlat, beforeNs, afterNs int64) string {
	if beforeFlat < 0 || afterFlat < 0 || beforeNs <= 0 || afterNs <= 0 {
		return "invalid"
	}
	if afterFlat < beforeFlat && afterNs < beforeNs {
		return "consistent_candidate"
	}
	if afterFlat > beforeFlat || afterNs > beforeNs {
		return "conflicting_signals"
	}
	return "no_change"
}
