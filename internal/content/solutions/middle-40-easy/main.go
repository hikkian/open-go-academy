package main

func EvidenceCount(required, passed int) int {
	if required < 0 || passed < 0 {
		return 0
	}
	if passed > required {
		return required
	}
	return passed
}
