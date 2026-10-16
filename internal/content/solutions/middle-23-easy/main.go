package main

func CacheVersionActionEasy(current, incoming int64) string {
	if current < 0 || incoming < 0 { return "invalid" }; if incoming > current { return "apply" }; return "ignore"
}
