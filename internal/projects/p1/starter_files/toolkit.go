package main

import "errors"

// ToolError is the target error type for short command errors. Return it from
// run so main can write one controlled stderr line and exit with status 2.
// TODO: declare ToolError with a Message field and implement Error() string.
// TODO: parse sum arguments, check int64 overflow, and print the total.
// TODO: inspect a JSON file containing an array of strings and print its count.
func run(args []string) error {
	return errors.New("TODO: implement sum and inspect")
}
