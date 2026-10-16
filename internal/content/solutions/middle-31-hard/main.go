package main

func PublicError(status int, hasInternalDetail bool) string {
	if status < 400 || status > 599 {
		return "invalid_status"
	}
	if hasInternalDetail {
		return "generic_error"
	}
	return "safe_error"
}
