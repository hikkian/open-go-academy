package main

func ReleaseDecision(acceptancePassed, rollbackReady, reviewerApproved bool) bool {
	return acceptancePassed && rollbackReady && reviewerApproved
}
