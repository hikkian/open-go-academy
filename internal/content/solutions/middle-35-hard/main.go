package main

func SafeExperiment(hasStopCondition, boundedScope, usesProductionData bool) bool {
	return hasStopCondition && boundedScope && !usesProductionData
}
