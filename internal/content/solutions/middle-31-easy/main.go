package main

func ProbeStatus(started, dependenciesReady bool) string {
	if !started {
		return "starting"
	}
	if !dependenciesReady {
		return "not_ready"
	}
	return "ready"
}
