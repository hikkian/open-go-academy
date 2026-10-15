package main

func ClampRange(value, low, high int) int {
 if low>high { return low }; if value<low { return low }; if value>high { return high }; return value
}
