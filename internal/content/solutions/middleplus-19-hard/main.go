package main

func SafeToRetry(stage string, committed bool, observed bool, retryable bool) bool {
	return stage == "publish" && committed && observed && retryable
}
