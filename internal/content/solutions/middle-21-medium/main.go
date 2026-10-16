package main

func CacheReadPolicyMedium(fresh, staleAllowed, backendAvailable bool) string {
	if fresh { return "serve_fresh" }; if staleAllowed { return "serve_stale" }; if backendAvailable { return "load" }; return "unavailable"
}
