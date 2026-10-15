package main

func LogFieldAllowed(name string) bool { switch name { case "request_id","method","status","duration_ms": return true; default: return false } }
