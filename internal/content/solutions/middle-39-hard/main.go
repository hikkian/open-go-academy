package main

func ReleaseEvidence(unit, race, failure, boundedLoad bool) bool {
	return unit && race && failure && boundedLoad
}
