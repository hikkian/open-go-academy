package main

func StorageFit(pattern, consistency string) string {
	if pattern == "" || consistency == "" {
		return "insufficient"
	}
	if pattern == "relational" && consistency == "transactional" {
		return "relational_candidate"
	}
	if pattern == "append" && consistency == "eventual" {
		return "log_candidate"
	}
	return "compare_options"
}
