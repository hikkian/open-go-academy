package main

func ProfileShare(sample, total int) int {
	if total <= 0 || total > int(^uint(0)>>1)/100 || sample < 0 || sample > total {
		return -1
	}
	return sample * 100 / total
}
