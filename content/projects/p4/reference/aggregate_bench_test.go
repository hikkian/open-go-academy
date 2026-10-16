package main

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

func benchmarkEvents() []Event {
	rng := rand.New(rand.NewSource(0x51ca1ab))
	services := make([]string, 64)
	regions := make([]string, 16)
	for i := range services {
		services[i] = fmt.Sprintf("service-%02d", i)
	}
	for i := range regions {
		regions[i] = fmt.Sprintf("region-%02d", i)
	}
	events := make([]Event, 4096)
	for i := range events {
		events[i] = Event{Service: services[rng.Intn(len(services))], Region: regions[rng.Intn(len(regions))], Bytes: uint64(rng.Intn(4096))}
	}
	return events
}

var benchmarkResult map[string]Totals

func assertBenchmarkResults(b *testing.B, events []Event) {
	b.Helper()
	baseline, err := AggregateBaseline(events)
	if err != nil {
		b.Fatalf("baseline correctness: %v", err)
	}
	optimized, err := Aggregate(events)
	if err != nil {
		b.Fatalf("optimized correctness: %v", err)
	}
	if !reflect.DeepEqual(baseline, optimized) {
		b.Fatalf("baseline and optimized results differ on benchmark dataset")
	}
}

func BenchmarkAggregateBaseline(b *testing.B) {
	events := benchmarkEvents()
	assertBenchmarkResults(b, events)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result, err := AggregateBaseline(events)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkAggregateOptimized(b *testing.B) {
	events := benchmarkEvents()
	assertBenchmarkResults(b, events)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result, err := Aggregate(events)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}
