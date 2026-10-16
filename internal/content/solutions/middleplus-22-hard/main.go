package main

func ContinueRollout(errorPercent int, threshold int, sampleCount int, minimumSample int, evidenceReady bool, rollbackReady bool) bool {
	if errorPercent < 0 || errorPercent > 100 || threshold < 0 || threshold > 100 || sampleCount < 0 || minimumSample <= 0 || !evidenceReady || !rollbackReady {
		return false
	}
	if sampleCount < minimumSample || errorPercent > threshold {
		return false
	}
	return true
}
