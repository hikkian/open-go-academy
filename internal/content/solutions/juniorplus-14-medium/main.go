package main

func CRUDStatus(m string, f bool) int {
	switch m {
	case "GET":
		if f {
			return 200
		}
		return 404
	case "POST":
		return 201
	case "PUT", "PATCH":
		if f {
			return 200
		}
		return 404
	case "DELETE":
		if f {
			return 204
		}
		return 404
	default:
		return 405
	}
}
