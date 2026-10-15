package main
import "strings"
func countWords(text string) map[string]int { out:=make(map[string]int); for _,w:=range strings.Fields(text) { out[w]++ }; return out }
