package main

import "strings"

func IsJSONSuccessResponse(s int, c string) bool {
	if s != 200 {
		return false
	}
	m := strings.ToLower(strings.TrimSpace(strings.SplitN(c, ";", 2)[0]))
	return m == "application/json"
}
