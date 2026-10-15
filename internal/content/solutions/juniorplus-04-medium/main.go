package main

func IsIdempotentMethod(m string) bool {
	switch m {
	case "GET", "HEAD", "PUT", "DELETE":
		return true
	}
	return false
}
