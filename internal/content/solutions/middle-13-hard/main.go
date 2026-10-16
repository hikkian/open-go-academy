package main

func IndexReadWriteDecision(matches, total, reads, writes int64) string {
	if matches < 0 || total < 0 || reads < 0 || writes < 0 || matches > total {
		return "invalid"
	}
	if total == 0 {
		return "no_rows"
	}
	if matches > (total-1)/4 {
		return "sequential"
	}
	if writes > reads {
		return "write_heavy"
	}
	return "index_candidate"
}
