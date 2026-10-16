package main

func AdmissionState(shuttingDown, ready bool) string {
	if shuttingDown {
		return "draining"
	}
	if !ready {
		return "not_ready"
	}
	return "accepting"
}
