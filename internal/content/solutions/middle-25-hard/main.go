package main

func RequestPathReadyHard(handlerBound, serviceBound, repositoryBound bool) bool {
	return handlerBound && serviceBound && repositoryBound
}
