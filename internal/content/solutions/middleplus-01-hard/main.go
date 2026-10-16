package main

func BaselineDecision(before, after, minImprovement int64) string {
	if before < 0 || after < 0 || minImprovement < 0 {
		return "invalid"
	}
	delta := before - after
	if delta < 0 {
		return "regression"
	}
	if delta >= minImprovement {
		return "candidate_improved"
	}
	return "below_threshold"
}
