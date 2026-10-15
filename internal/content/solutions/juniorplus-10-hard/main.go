package main

func LinkBelongsTo(requestedOwner, rowOwner string) bool { return requestedOwner != "" && rowOwner != "" && requestedOwner == rowOwner }
