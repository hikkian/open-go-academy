package main

func IncidentSignal(samples, failed int) string {
	if samples < 0 || failed < 0 || failed > samples {
		return "invalid"
	}
	if samples == 0 {
		return "unknown"
	}
	if failed > 0 {
		return "observed_failure"
	}
	return "no_failure_observed"
}
