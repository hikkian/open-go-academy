package main

func RequirementStatus(source, measurable string) string {
	if source == "" {
		return "unknown_source"
	}
	if measurable == "" {
		return "needs_measurement"
	}
	return "testable"
}
