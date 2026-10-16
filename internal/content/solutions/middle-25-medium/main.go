package main

func ServiceWiringMedium(repositoryBound, clockBound bool) string {
	if repositoryBound && clockBound { return "ready" }; if !repositoryBound && !clockBound { return "missing_repository_and_clock" }; if !repositoryBound { return "missing_repository" }; return "missing_clock"
}
