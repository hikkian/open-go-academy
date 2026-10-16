package main

func FeedbackKind(actionable, evidence bool) string {
	if !actionable {
		return "question"
	}
	if !evidence {
		return "needs_evidence"
	}
	return "suggestion"
}
