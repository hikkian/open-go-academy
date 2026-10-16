package main

func RetryDelayEasy(attempt, baseMS, maximumMS int64) int64 {
	if attempt <= 0 || baseMS < 0 || maximumMS < 0 { return 0 }; if baseMS > maximumMS { return maximumMS }; delay := baseMS; for i := int64(1); i < attempt && delay < maximumMS; i++ { if delay > maximumMS/2 { return maximumMS }; delay *= 2 }; return delay
}
