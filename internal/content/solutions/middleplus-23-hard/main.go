package main

func PostmortemReady(eventTimes []int, observed []bool, actionStatus string, ownerAssigned bool, dueDay int, verified bool) bool {
	if len(eventTimes) < 2 || len(eventTimes) != len(observed) || dueDay < 0 {
		return false
	}
	for i, t := range eventTimes {
		if t < 0 || (i > 0 && t < eventTimes[i-1]) || !observed[i] {
			return false
		}
	}
	return actionStatus == "closed" && ownerAssigned && dueDay > 0 && verified
}
