package main

func RequestDecision(bodyBytes, maxBytes int64, canceled bool) string {
	if canceled {
		return "canceled"
	}
	if bodyBytes < 0 || maxBytes <= 0 {
		return "invalid"
	}
	if bodyBytes > maxBytes {
		return "too_large"
	}
	return "continue"
}
