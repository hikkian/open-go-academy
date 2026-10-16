package main

func MergeReady(blockers, testsGreen, ownerApproved int) bool {
	return blockers == 0 && testsGreen == 1 && ownerApproved == 1
}
