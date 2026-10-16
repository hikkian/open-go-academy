package main

func MetricLabelAllowed(label string) bool {
	switch label {
	case "operation", "status_class", "result":
		return true
	default:
		return false
	}
}
