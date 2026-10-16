package main

import (
	"errors"
	"fmt"
)

var ErrOverflow = errors.New("event byte total overflows uint64")

type Event struct {
	Service string
	Region  string
	Bytes   uint64
}

type Totals struct {
	Count uint64
	Bytes uint64
}

type aggregateKey struct {
	service string
	region  string
}

// AggregateBaseline creates a formatted string for every event and grows the map
// from zero capacity. It is intentionally simple and correct, but allocation-heavy.
func AggregateBaseline(events []Event) (map[string]Totals, error) {
	out := make(map[string]Totals)
	for _, event := range events {
		key := encodeKey(event.Service, event.Region)
		current := out[key]
		if current.Count == ^uint64(0) || event.Bytes > ^uint64(0)-current.Bytes {
			return nil, ErrOverflow
		}
		current.Count++
		current.Bytes += event.Bytes
		out[key] = current
	}
	return out, nil
}

// AggregateKeyed avoids constructing a composite string for every event.
func AggregateKeyed(events []Event) (map[aggregateKey]Totals, error) {
	capacity := len(events)
	if capacity > 256 {
		capacity = 256
	}
	out := make(map[aggregateKey]Totals, capacity)
	for _, event := range events {
		key := aggregateKey{service: event.Service, region: event.Region}
		current := out[key]
		if current.Count == ^uint64(0) || event.Bytes > ^uint64(0)-current.Bytes {
			return nil, ErrOverflow
		}
		current.Count++
		current.Bytes += event.Bytes
		out[key] = current
	}
	return out, nil
}

func stringify(keyed map[aggregateKey]Totals) map[string]Totals {
	out := make(map[string]Totals, len(keyed))
	for key, total := range keyed {
		out[encodeKey(key.service, key.region)] = total
	}
	return out
}

// Length-prefixing keeps keys unambiguous even when either field contains NUL
// or digits that resemble a separator.
func encodeKey(service, region string) string {
	return fmt.Sprintf("%d:%s%d:%s", len(service), service, len(region), region)
}

func Aggregate(events []Event) (map[string]Totals, error) {
	keyed, err := AggregateKeyed(events)
	if err != nil {
		return nil, err
	}
	return stringify(keyed), nil
}
