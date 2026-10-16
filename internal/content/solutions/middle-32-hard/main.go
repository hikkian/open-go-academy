package main

func SignalDecision(requests, failures int) string {
	if requests < 0 || failures < 0 || failures > requests {
		return "invalid"
	}
	if requests == 0 {
		return "no_sample"
	}
	if failures > 0 {
		return "errors_present"
	}
	return "no_errors_observed"
}
