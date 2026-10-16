package main

func DrainDeadline(nowMillis, graceMillis int64) int64 {
	if nowMillis < 0 || graceMillis < 0 {
		return -1
	}
	if nowMillis > int64(^uint64(0)>>1)-graceMillis {
		return -1
	}
	return nowMillis + graceMillis
}
