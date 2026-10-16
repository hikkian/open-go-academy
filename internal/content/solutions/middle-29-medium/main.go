package main

func EffectiveTimeout(parentMillis, handlerMillis int64) int64 {
	if parentMillis <= 0 || handlerMillis <= 0 {
		return 0
	}
	if parentMillis < handlerMillis {
		return parentMillis
	}
	return handlerMillis
}
