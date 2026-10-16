package main

func ReviewPriority(severity int, dataBoundary bool) string {
	if severity < 0 {
		return "invalid"
	}
	if dataBoundary {
		return "blocking"
	}
	if severity >= 3 {
		return "high"
	}
	return "normal"
}
