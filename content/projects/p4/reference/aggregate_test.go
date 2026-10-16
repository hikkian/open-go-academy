package main

import (
	"math"
	"reflect"
	"testing"
)

func TestAggregateVariantsAgree(t *testing.T) {
	events := []Event{
		{Service: "api", Region: "west", Bytes: 12},
		{Service: "api", Region: "west", Bytes: 0},
		{Service: "", Region: "", Bytes: 8},
		{Service: "worker", Region: "east", Bytes: 5},
	}
	want := map[string]Totals{"3:api4:west": {Count: 2, Bytes: 12}, "0:0:": {Count: 1, Bytes: 8}, "6:worker4:east": {Count: 1, Bytes: 5}}
	for name, aggregate := range map[string]func([]Event) (map[string]Totals, error){
		"baseline":  AggregateBaseline,
		"optimized": Aggregate,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := aggregate(events)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, %v; want %v", got, err, want)
			}
		})
	}
}

func TestAggregateCompositeKeysAreUnambiguous(t *testing.T) {
	events := []Event{{Service: "a\x00b", Region: "c"}, {Service: "a", Region: "b\x00c"}}
	got, err := Aggregate(events)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %v, %v; want two distinct groups", got, err)
	}
}

func TestAggregateEmpty(t *testing.T) {
	got, err := Aggregate(nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("Aggregate(nil) = %v, %v; want empty result", got, err)
	}
}

func TestAggregateOverflowReturnsNoPartialResult(t *testing.T) {
	events := []Event{{Service: "api", Region: "west", Bytes: math.MaxUint64}, {Service: "api", Region: "west", Bytes: 1}}
	for name, aggregate := range map[string]func([]Event) (map[string]Totals, error){"baseline": AggregateBaseline, "optimized": Aggregate} {
		t.Run(name, func(t *testing.T) {
			got, err := aggregate(events)
			if err != ErrOverflow || got != nil {
				t.Fatalf("got %v, %v; want nil, ErrOverflow", got, err)
			}
		})
	}
}
