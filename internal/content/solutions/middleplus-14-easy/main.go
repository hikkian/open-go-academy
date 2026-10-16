package main

func Admission(depth int, limit int) string {
	if depth < 0 || limit <= 0 {
		return "invalid"
	}
	if depth >= limit {
		return "full"
	}
	return "admit"
}
