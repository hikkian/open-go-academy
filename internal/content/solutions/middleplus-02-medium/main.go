package main

func ValidBenchmarkResult(nsPerOp, bytesPerOp, allocsPerOp int64) bool {
	return nsPerOp > 0 && bytesPerOp >= 0 && allocsPerOp >= 0
}
