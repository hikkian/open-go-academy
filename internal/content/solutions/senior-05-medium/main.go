package main

func ErrorClass(code int) string { switch code { case 0: return "ok"; case 1: return "invalid"; case 2: return "temporary"; default: return "unknown" } }
