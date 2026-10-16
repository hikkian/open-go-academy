package main

func WorkerTurn(arrivalA int, arrivalB int, waitA int, waitB int, maxWait int) string {
	if arrivalA < 0 || arrivalB < 0 || waitA < 0 || waitB < 0 || maxWait < 0 {
		return "invalid"
	}
	if waitA >= maxWait && waitB >= maxWait {
		if waitA >= waitB {
			return "A"
		}
		return "B"
	}
	if waitA >= maxWait {
		return "A"
	}
	if waitB >= maxWait {
		return "B"
	}
	if arrivalA <= arrivalB {
		return "A"
	}
	return "B"
}
