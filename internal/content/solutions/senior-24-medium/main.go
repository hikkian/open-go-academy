package main

func DecisionReview(triggerObserved, evidenceUpdated, ownerAssigned bool) string {
	if !triggerObserved {
		return "keep_watch"
	}
	if !evidenceUpdated {
		return "gather_evidence"
	}
	if !ownerAssigned {
		return "assign_owner"
	}
	return "revisit_decision"
}
