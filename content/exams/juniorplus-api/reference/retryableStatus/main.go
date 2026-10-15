package main

func RetryableStatus(status int) bool { return status==429 || (status>=500 && status<=599) }
