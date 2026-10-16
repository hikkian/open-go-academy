package main

func ShutdownAction(shuttingDown bool, active int, graceExpired bool) string {
	if !shuttingDown {
		return "serve"
	}
	if active < 0 {
		return "invalid"
	}
	if active == 0 {
		return "stopped"
	}
	if graceExpired {
		return "cancel_remaining"
	}
	return "wait"
}
