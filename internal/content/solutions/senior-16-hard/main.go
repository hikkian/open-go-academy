package main

func ScaleEvidence(sameWorkload, stateMeasured, failureTested bool) bool {
	return sameWorkload && stateMeasured && failureTested
}
