package main

import "unicode/utf8"

func countRunes(text string) int { return utf8.RuneCountInString(text) }
