package main

func ProtectCritical(offered int, capacity int, critical bool, deadlineOK bool, queueAge int, maxQueueAge int) bool {
	if offered < 0 || capacity <= 0 || queueAge < 0 || maxQueueAge < 0 {
		return false
	}
	if offered <= capacity {
		return true
	}
	return critical && deadlineOK && queueAge <= maxQueueAge
}
