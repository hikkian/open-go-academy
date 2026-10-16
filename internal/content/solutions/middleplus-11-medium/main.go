package main

func ReplicaSafeForRead(wrote, replicaFresh, staleTolerant bool) bool {
	return !wrote && replicaFresh && staleTolerant
}
