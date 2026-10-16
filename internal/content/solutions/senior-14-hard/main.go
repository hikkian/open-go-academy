package main

func StorageDecision(queryFit, durabilityFit, restoreTested bool) bool {
	return queryFit && durabilityFit && restoreTested
}
