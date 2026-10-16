package main

func IndexScanCandidate(matches, total int64) string {
	if matches < 0 || total < 0 || matches > total {
		return "invalid"
	}
	if total == 0 {
		return "no_rows"
	}
	if matches <= (total-1)/4 {
		return "index_candidate"
	}
	return "sequential"
}
