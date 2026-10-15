package main

func MethodStatus(m string) int {
	switch m {
	case "GET", "HEAD":
		return 200
	case "POST":
		return 201
	case "PUT", "PATCH":
		return 200
	case "DELETE":
		return 204
	default:
		return 405
	}
}
