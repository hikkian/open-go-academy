package main

func MigrationPhase(expanded, backfillVerified, consumersMoved bool) string {
	if !expanded {
		return "expand"
	}
	if !backfillVerified {
		return "backfill"
	}
	if !consumersMoved {
		return "cutover_ready"
	}
	return "contract_candidate"
}
