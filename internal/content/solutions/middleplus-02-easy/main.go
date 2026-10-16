package main

func MetricLabel(metric string) string {
	switch metric {
	case "time":
		return "ns_per_op"
	case "bytes":
		return "bytes_per_op"
	case "allocs":
		return "allocs_per_op"
	default:
		return "unknown"
	}
}
