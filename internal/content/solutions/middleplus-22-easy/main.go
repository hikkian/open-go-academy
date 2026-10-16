package main

func RolloutAction(errorPercent int, threshold int, sampleCount int, minimumSample int, evidenceReady bool) string {
	if errorPercent < 0 || errorPercent > 100 || threshold < 0 || threshold > 100 || sampleCount < 0 || minimumSample <= 0 {
		return "invalid"
	}
	if !evidenceReady || sampleCount < minimumSample {
		return "hold"
	}
	if errorPercent > threshold {
		return "rollback"
	}
	return "expand"
}
