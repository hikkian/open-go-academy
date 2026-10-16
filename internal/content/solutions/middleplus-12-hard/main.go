package main

func CacheReadPolicy(writeSucceeded, invalidationSucceeded, versionMatches bool) string {
	if writeSucceeded && invalidationSucceeded && versionMatches {
		return "cache"
	}
	return "source"
}
