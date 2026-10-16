package main

func CanStart(dbConfigured, repositoryWired, handlerWired bool) bool {
	return dbConfigured && repositoryWired && handlerWired
}
