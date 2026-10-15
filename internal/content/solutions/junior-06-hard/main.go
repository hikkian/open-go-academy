package main

import "fmt"

type InputError struct { Code int }

func (e *InputError) Error() string { return "invalid input" }

func describeFailure(code int, cause error) error {
	switch code {
	case 1:
		return fmt.Errorf("lookup: %w", cause)
	case 2:
		return fmt.Errorf("input: %w", &InputError{Code: 2})
	default:
		return nil
	}
}
