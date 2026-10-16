package main

func BackoffMillis(attempt, baseMillis, maxMillis int) int {
	if attempt < 0 || baseMillis < 0 || maxMillis < 0 {
		return 0
	}
	// A zero base never grows. Return immediately instead of looping up to an
	// attacker-controlled (or simply mistaken) attempt count.
	if baseMillis == 0 || maxMillis == 0 {
		return 0
	}
	delay := baseMillis
	for i := 0; i < attempt && delay < maxMillis; i++ {
		if delay > maxMillis/2 {
			return maxMillis
		}
		delay *= 2
	}
	if delay > maxMillis {
		return maxMillis
	}
	return delay
}
