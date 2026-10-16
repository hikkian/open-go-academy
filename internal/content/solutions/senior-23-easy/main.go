package main

func EvidenceStrength(reproduced, independentReview, boundedClaim bool) string {
	if !boundedClaim {
		return "overclaim"
	}
	if !reproduced {
		return "not_reproduced"
	}
	if !independentReview {
		return "needs_review"
	}
	return "reviewed_evidence"
}
