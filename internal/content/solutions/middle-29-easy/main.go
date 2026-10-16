package main

func BodyAllowed(bodyBytes, maxBytes int64) bool {
	return bodyBytes >= 0 && maxBytes > 0 && bodyBytes <= maxBytes
}
