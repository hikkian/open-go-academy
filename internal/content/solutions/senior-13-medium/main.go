package main

func DecisionRecord(option, evidence, revisit string) string {
	if option == "" {
		return "missing_option"
	}
	if evidence == "" {
		return "assumption"
	}
	if revisit == "" {
		return "no_revisit_trigger"
	}
	return "recorded"
}
