package main

func Enqueue(depth, capacity int64) string { if depth < 0 || capacity <= 0 { return "invalid" }; if depth >= capacity { return "reject" }; return "accept" }
