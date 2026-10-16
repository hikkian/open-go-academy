package main

func RedisRoleEasy(durabilityRequired, replayable bool) string {
	if durabilityRequired { return "needs_review" }; return "cache"
}
