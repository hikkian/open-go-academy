package main

func ContractVersion(version int, supported int) string {
	if version <= 0 || supported <= 0 {
		return "invalid"
	}
	if version > supported {
		return "unsupported"
	}
	return "accepted"
}
