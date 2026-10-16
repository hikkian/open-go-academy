package main

func CacheFailurePolicyHard(fresh, staleAllowed, backendAvailable, staleData bool) string {
	if fresh { return "serve_fresh" }; if staleAllowed && staleData { return "serve_stale" }; if backendAvailable { return "load" }; return "unavailable"
}
