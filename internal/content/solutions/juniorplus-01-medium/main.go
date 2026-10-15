package main

import "fmt"

func RequestSummary(method, path, host string, headerCount, bodyBytes int) string {
	if method == "" || path == "" || host == "" || headerCount < 0 || bodyBytes < 0 {
		return "invalid"
	}
	return fmt.Sprintf("%s %s host=%s headers=%d body=%d", method, path, host, headerCount, bodyBytes)
}
