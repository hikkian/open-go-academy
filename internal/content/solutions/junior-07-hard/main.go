package main

import (
	"io"
)

func readAll(r io.Reader) (string, error) { b, e := io.ReadAll(r); return string(b), e }
