package main

func CacheEventActionMedium(current, incoming int64, deleted bool) string {
	if current < 0 || incoming < 0 { return "invalid" }; if incoming <= current { return "ignore" }; if deleted { return "invalidate" }; return "refresh"
}
