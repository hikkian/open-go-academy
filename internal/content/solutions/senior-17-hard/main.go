package main

func CutoverAllowed(shadowMatched, rollbackTested, ownerApproved bool) bool {
	return shadowMatched && rollbackTested && ownerApproved
}
