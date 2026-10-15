package main

func AssessmentDisposition(automaticPassed,securityGatePassed,manualReviewComplete bool) string { if !securityGatePassed{return "security_gate_failed"};if !automaticPassed{return "automatic_checks_incomplete"};if !manualReviewComplete{return "manual_review_required"};return "reviewed_evidence" }
