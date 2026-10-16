package main

func AttributeAllowed(key string) bool {
	switch key {
	case "route_template", "status_class", "operation":
		return true
	default:
		return false
	}
}
