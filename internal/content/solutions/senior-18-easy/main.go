package main

func SignalCoverage(question, source, window string) string {
	if question == "" {
		return "missing_question"
	}
	if source == "" {
		return "missing_source"
	}
	if window == "" {
		return "missing_window"
	}
	return "defined"
}
