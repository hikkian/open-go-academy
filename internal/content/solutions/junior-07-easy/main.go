package main

import (
	"io"
)

func canRead(r io.Reader) bool { return r != nil }
