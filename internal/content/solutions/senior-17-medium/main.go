package main

func RollbackWindow(dataCompatible, oldVersionAvailable bool) string {
	if !oldVersionAvailable {
		return "no_binary_rollback"
	}
	if !dataCompatible {
		return "data_blocked"
	}
	return "rollback_available"
}
