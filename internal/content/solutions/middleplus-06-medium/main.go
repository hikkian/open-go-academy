package main

func SameProfileWindow(beforeSeconds, afterSeconds int, sameWorkload bool) bool {
	return beforeSeconds > 0 && beforeSeconds == afterSeconds && sameWorkload
}
