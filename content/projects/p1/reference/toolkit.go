package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
)

// ToolError is a concise, user-facing command error. main writes it to stderr
// and exits with status 2; it deliberately carries no stack trace.
type ToolError struct {
	Message string
}

func (e *ToolError) Error() string { return e.Message }

func toolError(message string) error { return &ToolError{Message: message} }

func run(args []string) error {
	if len(args) == 0 {
		return toolError("usage: toolkit sum N... | inspect path")
	}
	switch args[0] {
	case "sum":
		if len(args) < 2 {
			return toolError("usage: toolkit sum N...")
		}
		var total int64
		for _, arg := range args[1:] {
			n, err := strconv.ParseInt(arg, 10, 64)
			if err != nil {
				return toolError(fmt.Sprintf("invalid integer %q", arg))
			}
			if (n > 0 && total > math.MaxInt64-n) || (n < 0 && total < math.MinInt64-n) {
				return toolError("sum overflows int64")
			}
			total += n
		}
		fmt.Println(total)
		return nil
	case "inspect":
		if len(args) != 2 {
			return toolError("usage: toolkit inspect path")
		}
		data, err := os.ReadFile(args[1])
		if err != nil {
			return toolError(fmt.Sprintf("cannot read %q", args[1]))
		}
		var values []string
		if err := json.Unmarshal(data, &values); err != nil || values == nil {
			return toolError("input must be a JSON array of strings")
		}
		fmt.Println(len(values))
		return nil
	default:
		return toolError(fmt.Sprintf("unknown command %q; use sum or inspect", args[0]))
	}
}
