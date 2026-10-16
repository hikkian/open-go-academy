package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	q := NewQueue()
	if _, err := q.Enqueue(Event{ID: "evt-1", TenantID: "tenant-demo", Payload: []byte("order.updated")}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	claim, err := q.Claim("worker-a", time.Unix(100, 0), 30*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if claim == nil {
		fmt.Fprintln(os.Stderr, "no claim available")
		os.Exit(1)
		return
	}
	fmt.Printf("claimed %s with fencing token %d; next step is to make the downstream effect idempotent\n", claim.Event.ID, claim.FenceToken)
}
