package main

func TraceAttributeAllowed(key, value string) bool {
	switch key {
	case "route":
		return value == "/v1/tasks/{taskID}" || value == "/v1/projects/{projectID}/tasks/{taskID}"
	case "method":
		switch value {
		case "GET", "POST", "PATCH", "DELETE":
			return true
		default:
			return false
		}
	case "status_code":
		if len(value) != 3 || value[0] < '1' || value[0] > '5' {
			return false
		}
		for i := 1; i < len(value); i++ {
			if value[i] < '0' || value[i] > '9' {
				return false
			}
		}
		return true
	default:
		return false
	}
}
