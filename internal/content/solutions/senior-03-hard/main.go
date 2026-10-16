package main

func Admission(queue, limit int64) string { if queue < 0 || limit <= 0 { return "invalid" }; if queue >= limit { return "reject" }; return "accept" }
