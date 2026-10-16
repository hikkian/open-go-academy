package main

func CacheEntryValid(ageSeconds, ttlSeconds int64, versionMatches bool) bool {
	return ageSeconds >= 0 && ttlSeconds > 0 && ageSeconds < ttlSeconds && versionMatches
}
