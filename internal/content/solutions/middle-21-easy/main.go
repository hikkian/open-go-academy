package main

func CacheTTLEasy(seconds int64) int64 {
	if seconds <= 0 { return 0 }; if seconds > 3600 { return 3600 }; return seconds
}
