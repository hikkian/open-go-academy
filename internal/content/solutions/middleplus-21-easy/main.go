package main

func SLOBand(total int, failed int) string {
	if total <= 0 || failed < 0 || failed > total { return "invalid" }; if failed == 0 { return "within" }; return "errors"
}
