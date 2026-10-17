package main

import (
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

type retryInput struct {
	Available, Capacity, Elapsed, Rate, Cost int64
}

func (retryInput) Generate(r *rand.Rand, _ int) reflect.Value {
	const max = int64(^uint64(0) >> 1)
	capacity := int64(r.Uint64() & uint64(max-1))
	available := int64(r.Uint64() & uint64(capacity))
	cost := int64(r.Uint64() & uint64(max))
	if cost == 0 {
		cost = 1
	}
	input := retryInput{
		Available: available,
		Capacity:  capacity,
		Elapsed:   int64(r.Uint64() & uint64(max)),
		Rate:      int64(r.Uint64() & uint64(max)),
		Cost:      cost,
	}
	// Generate each invalid dimension directly, keeping the other fields valid.
	switch r.Intn(7) {
	case 1:
		input.Capacity = -1
	case 2:
		input.Available = -1
	case 3:
		input.Available = input.Capacity + 1
	case 4:
		input.Elapsed = -1
	case 5:
		input.Rate = -1
	case 6:
		input.Cost = 0
	}
	return reflect.ValueOf(input)
}

// big.Int is deliberately used as an independent arithmetic oracle: it computes
// the unsaturated product exactly, then applies the contract's capacity bound.
func retryOracle(x retryInput) int64 {
	if x.Capacity < 0 || x.Available < 0 || x.Available > x.Capacity || x.Elapsed < 0 || x.Rate < 0 || x.Cost <= 0 || x.Cost > x.Capacity {
		return -1
	}
	available := new(big.Int).SetInt64(x.Available)
	refill := new(big.Int).Mul(big.NewInt(x.Elapsed), big.NewInt(x.Rate))
	available.Add(available, refill)
	capacity := big.NewInt(x.Capacity)
	if available.Cmp(capacity) > 0 {
		available.Set(capacity)
	}
	if available.Cmp(big.NewInt(x.Cost)) >= 0 {
		return 0
	}
	if x.Rate == 0 {
		return -1
	}
	deficit := new(big.Int).Sub(big.NewInt(x.Cost), available)
	rate := big.NewInt(x.Rate)
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(deficit, rate, rem)
	if rem.Sign() != 0 {
		q.Add(q, big.NewInt(1))
	}
	return q.Int64()
}

func TestRetryAfterSecondsHardMatchesBigIntOracle(t *testing.T) {
	cfg := quick.Config{MaxCount: 5000, Rand: rand.New(rand.NewSource(220018))}
	err := quick.Check(func(x retryInput) bool {
		got := RetryAfterSecondsHard(x.Available, x.Capacity, x.Elapsed, x.Rate, x.Cost)
		return got == retryOracle(x)
	}, &cfg)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRetryAfterSecondsHardInt64Boundaries(t *testing.T) {
	const max = int64(^uint64(0) >> 1)
	cases := []struct {
		name string
		in   retryInput
		want int64
	}{
		{"elapsed refills before wait", retryInput{2, 10, 1, 2, 5}, 1},
		{"elapsed saturates before wait", retryInput{2, 10, 2, 2, 5}, 0},
		{"cost above capacity is unreachable", retryInput{3, 4, max, 1, 5}, -1},
		{"ceil near maximum does not overflow", retryInput{0, max, 0, 2, max}, 4611686018427387904},
		{"maximum rate and deficit", retryInput{0, max, 0, max, max}, 1},
		{"maximum elapsed saturates safely", retryInput{max - 7, max, max, max, max}, 0},
		{"zero refill and insufficient tokens", retryInput{0, max, 0, 0, 1}, -1},
		{"negative capacity", retryInput{0, -1, 0, 1, 1}, -1},
		{"negative available", retryInput{-1, 10, 0, 1, 1}, -1},
		{"available above capacity", retryInput{11, 10, 0, 1, 1}, -1},
		{"negative elapsed", retryInput{0, 10, -1, 1, 1}, -1},
		{"negative refill rate", retryInput{0, 10, 0, -1, 1}, -1},
		{"zero cost", retryInput{0, 10, 0, 1, 0}, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RetryAfterSecondsHard(tc.in.Available, tc.in.Capacity, tc.in.Elapsed, tc.in.Rate, tc.in.Cost)
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
