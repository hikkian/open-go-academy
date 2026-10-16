package main

func ContractReady(hasOwner, hasErrors, hasVersionPolicy bool) bool {
	return hasOwner && hasErrors && hasVersionPolicy
}
