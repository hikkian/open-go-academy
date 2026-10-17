package main

func RetryAfterSecondsHard(available, capacity, elapsedSeconds, refillPerSecond, cost int64) int64 {
	if capacity < 0 || available < 0 || available > capacity || elapsedSeconds < 0 || refillPerSecond < 0 || cost <= 0 {
		return -1
	}
	// A bucket can never accumulate more than capacity, so this request is unreachable.
	if cost > capacity {
		return -1
	}

	// Saturate before multiplying: once elapsed refill reaches the remaining room,
	// the exact product is irrelevant and might overflow int64.
	room := capacity - available
	if refillPerSecond > 0 && elapsedSeconds >= ceilDivNonnegative(room, refillPerSecond) {
		available = capacity
	} else {
		available += elapsedSeconds * refillPerSecond
	}
	if available >= cost {
		return 0
	}
	if refillPerSecond == 0 {
		return -1
	}
	return ceilDivNonnegative(cost-available, refillPerSecond)
}

func ceilDivNonnegative(n, d int64) int64 {
	q := n / d
	if n%d != 0 {
		q++
	}
	return q
}
