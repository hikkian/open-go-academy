package main

func CanaryDecision(errorPercent int, threshold int, sampleCount int, minimumSample int, evidenceReady bool, rollbackReady bool) string {
	if errorPercent < 0 || errorPercent > 100 || threshold < 0 || threshold > 100 || sampleCount < 0 || minimumSample <= 0 {
		return "invalid"
	}
	if !evidenceReady || sampleCount < minimumSample {
		return "insufficient"
	}
	if errorPercent > threshold {
		if rollbackReady {
			return "rollback"
		}
		return "stop"
	}
	return "expand"
}
