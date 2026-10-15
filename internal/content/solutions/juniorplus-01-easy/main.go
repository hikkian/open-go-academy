package main

func ClassifyStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "success"
	case code >= 300 && code < 400:
		return "redirect"
	case code >= 400 && code < 500:
		return "client-error"
	case code >= 500 && code < 600:
		return "server-error"
	default:
		return "other"
	}
}
