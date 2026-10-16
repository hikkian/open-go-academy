package main

import (
	"math"
	"testing"
)

func TestStarterCompilesAndMarksWorkIncomplete(t *testing.T) {
	got, err := aggregateBaseline([]Event{{Service: "api", Region: "west", Bytes: 9}, {Service: "api", Region: "west", Bytes: 2}})
	if err != nil || got["3:api4:west"] != (Totals{Count: 2, Bytes: 11}) {
		t.Fatalf("baseline = %v, %v", got, err)
	}
	// This only verifies the provided baseline. Add optimized equivalence and
	// benchmark tests before considering the project complete.
}

func TestBaselineOverflow(t *testing.T) {
	got, err := aggregateBaseline([]Event{{Bytes: math.MaxUint64}, {Bytes: 1}})
	if err != errOverflow || got != nil {
		t.Fatalf("got %v, %v; want nil and overflow", got, err)
	}
}
