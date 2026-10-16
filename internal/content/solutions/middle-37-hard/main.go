package main

func MutationAllowed(authenticated, sameTenant, canWrite, validInput bool) bool {
	return authenticated && sameTenant && canWrite && validInput
}
