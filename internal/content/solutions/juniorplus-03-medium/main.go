package main

func TLSBoundary(scheme string) string {
	if scheme == "https" {
		return "tls-required"
	}
	if scheme == "http" {
		return "tls-not-used"
	}
	return "unsupported-scheme"
}
