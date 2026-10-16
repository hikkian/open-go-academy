package main

func CacheAction(writeSucceeded, invalidationSucceeded bool) string {
	if !writeSucceeded {
		return "no_write"
	}
	if !invalidationSucceeded {
		return "bypass_or_retry"
	}
	return "cache_consistent"
}
