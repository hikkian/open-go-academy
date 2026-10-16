package main

func RetryDelay(attempt int, baseMillis int, capMillis int) int {
	if attempt < 0 || baseMillis < 0 || capMillis < 0 {
		return 0
	}
	delay := baseMillis
	for n := 0; n < attempt && delay < capMillis; n++ {
		if delay > capMillis/2 {
			return capMillis
		}
		delay *= 2
	}
	if delay > capMillis {
		return capMillis
	}
	return delay
}
