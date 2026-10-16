package main

func RefilledTokensMedium(available, capacity, elapsedSeconds, refillPerSecond, cost int64) int64 {
	if capacity < 0 || available < 0 || available > capacity || elapsedSeconds < 0 || refillPerSecond < 0 || cost <= 0 { return -1 }; refill := capacity - available; if refillPerSecond > 0 && elapsedSeconds > refill/refillPerSecond { available = capacity } else { available += elapsedSeconds * refillPerSecond }; if available > capacity { available = capacity }; if available < cost { return -1 }; return available - cost
}
