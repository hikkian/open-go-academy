package main

import "strings"

func classifyDiagnostic(message string) string {
 s := strings.ToLower(message)
 if strings.Contains(s, "command not found") { return "tool" }; if strings.Contains(s, "no such file") || strings.Contains(s, "cannot find") { return "path" }; return "source"
}
