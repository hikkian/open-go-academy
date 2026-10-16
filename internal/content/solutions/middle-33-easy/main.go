package main

func ErrorBand(total, failed int) string {
	if total < 0 || failed < 0 || failed > total {
		return "invalid"
	}
	if total == 0 {
		return "no_sample"
	}
	if failed == 0 {
		return "no_errors"
	}
	return "errors"
}
