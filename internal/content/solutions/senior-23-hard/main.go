package main

func DefenseStatus(openRisks, reviewerSigned int) string {
	if openRisks < 0 || reviewerSigned < 0 || reviewerSigned > 1 {
		return "invalid"
	}
	if reviewerSigned == 0 {
		return "not_reviewed"
	}
	if openRisks > 0 {
		return "reviewed_with_risks"
	}
	return "reviewed_no_listed_risks"
}
