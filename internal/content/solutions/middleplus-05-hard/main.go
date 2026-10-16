package main

func HeapChange(before, after int64, gcCompleted bool) string {
	if before < 0 || after < 0 {
		return "invalid"
	}
	if !gcCompleted {
		return "gc_not_completed"
	}
	if after < before {
		return "lower_after_gc"
	}
	if after > before {
		return "higher_after_gc"
	}
	return "unchanged"
}
