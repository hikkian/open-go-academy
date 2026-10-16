package main

import (
	"errors"
	"fmt"
)

var errOverflow = errors.New("event byte total overflows uint64")

type Event struct {
	Service string
	Region  string
	Bytes   uint64
}

type Totals struct {
	Count uint64
	Bytes uint64
}

// Aggregate is a correct but deliberately allocation-heavy baseline.
// Replace it only after recording a benchmark and profile; keep this baseline
// available under a separate name so before/after remains reproducible.
func Aggregate(events []Event) (map[string]Totals, error) {
	return aggregateBaseline(events)
}

func aggregateBaseline(events []Event) (map[string]Totals, error) {
	out := make(map[string]Totals)
	for _, event := range events {
		key := fmt.Sprintf("%d:%s%d:%s", len(event.Service), event.Service, len(event.Region), event.Region)
		current := out[key]
		if current.Count == ^uint64(0) || event.Bytes > ^uint64(0)-current.Bytes {
			return nil, errOverflow
		}
		current.Count++
		current.Bytes += event.Bytes
		out[key] = current
	}
	return out, nil
}

// AggregateBaseline keeps the same public contract as the reference solution
// so trusted tests can compare the submitted implementation with its baseline.
func AggregateBaseline(events []Event) (map[string]Totals, error) {
	return aggregateBaseline(events)
}

func aggregateOptimized(events []Event) (map[string]Totals, error) {
	// TODO: student stage 2 — implement and justify a measured optimization.
	return nil, nil
}
