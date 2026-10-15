package main

func RetryableSQLState(sqlState string) bool { return sqlState == "40001" || sqlState == "40P01" }
