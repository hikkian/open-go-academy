package main

func ComparableBaseline(validWorkload, sameUnit, sameEnvironment bool) bool {
	return validWorkload && sameUnit && sameEnvironment
}
