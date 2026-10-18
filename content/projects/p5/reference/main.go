package main

import (
	"fmt"
	"time"
)

func main() {
	q := NewQueue()
	if _, err := q.Enqueue(Event{ID: "evt-1", TenantID: "tenant-demo", Payload: []byte("order.updated")}); err != nil {
		panic(err)
	}
	claim, err := q.Claim("worker-a", time.Unix(100, 0), 30*time.Second)
	if err != nil {
		panic(err)
	}
	if claim == nil {
		panic("no claim available")
	}
	fmt.Printf("claimed %s with fencing token %d; next step is to make the downstream effect idempotent\n", claim.Event.ID, claim.FenceToken)
}
