package main

func PoolStateEasy(inUse int64, maximum int64) string {
	if inUse < 0 || maximum <= 0 || inUse > maximum { return "invalid" }
	if inUse == maximum { return "wait" }
	return "available"
}
