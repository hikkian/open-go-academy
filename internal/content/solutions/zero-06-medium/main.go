package main

import "strings"

func nextSafeStep(message string) string {
 s := strings.ToLower(message)
 if strings.Contains(s, "command not found") { return "check_install" }; if strings.Contains(s, "no such file") { return "check_path" }; if strings.Contains(s, "syntax error") || strings.Contains(s, "undefined:") { return "read_first_error" }; return "save_and_rerun"
}
