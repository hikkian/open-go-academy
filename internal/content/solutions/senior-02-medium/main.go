package main

func ErrorClass(total, failures int64) string { if total <= 0 || failures < 0 || failures > total { return "invalid" }; if failures == 0 { return "clean-sample" }; return "observed-errors" }
