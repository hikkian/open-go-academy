package main

func IdempotencyResult(key string, alreadyApplied bool) string {
	if key == "" {
		return "missing_key"
	}
	if alreadyApplied {
		return "duplicate_ack"
	}
	return "apply_once"
}
