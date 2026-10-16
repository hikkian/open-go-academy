package main

func PoolAcquireMedium(inUse int64, maximum int64, waiters int64, maxWaiters int64, remainingMs int64) string {
	if inUse < 0 || maximum <= 0 || inUse > maximum || waiters < 0 || maxWaiters < 0 || remainingMs < 0 { return "invalid" }
	if inUse < maximum { return "acquire" }
	if remainingMs == 0 { return "timeout" }
	if waiters >= maxWaiters { return "reject" }
	return "wait"
}
