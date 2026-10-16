package main

func HTTPStatusForDomainErrorMedium(code string) int {
	switch code { case "invalid": return 400; case "conflict": return 409; case "not_found": return 404; default: return 500 }
}
