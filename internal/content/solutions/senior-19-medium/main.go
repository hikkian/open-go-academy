package main

func CommandAction(commanderAssigned, mitigationKnown bool) string {
	if !commanderAssigned {
		return "assign_commander"
	}
	if !mitigationKnown {
		return "stabilize_and_gather"
	}
	return "execute_mitigation"
}
