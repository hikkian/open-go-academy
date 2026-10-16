package main

func DecisionReview(triggerObserved, newEvidenceAvailable, evidenceVerified, ownerAssigned bool) string {
	if !triggerObserved {
		return "keep_watch"
	}
	if !newEvidenceAvailable {
		return "gather_evidence"
	}
	if !evidenceVerified {
		return "verify_evidence"
	}
	if !ownerAssigned {
		return "assign_owner"
	}
	return "revisit_decision"
}
