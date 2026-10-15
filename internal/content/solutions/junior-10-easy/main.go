package main

func hasFlag(args []string, flag string) bool {
	if flag == "" {
		return false
	}
	for _, v := range args {
		if v == flag {
			return true
		}
	}
	return false
}
