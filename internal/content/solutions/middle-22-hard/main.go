package main

func RetryAfterSecondsHard(available, capacity, elapsedSeconds, refillPerSecond, cost int64) int64 {
	if capacity < 0 || available < 0 || available > capacity || elapsedSeconds < 0 || refillPerSecond < 0 || cost <= 0 { return -1 }; if available >= cost { return 0 }; if refillPerSecond == 0 { return -1 }; deficit := cost - available; return (deficit + refillPerSecond - 1) / refillPerSecond
}
