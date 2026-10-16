package main

func ValidationResultEasy(valid bool) string {
	if !valid { return "invalid" }; return "ready_to_write"
}
