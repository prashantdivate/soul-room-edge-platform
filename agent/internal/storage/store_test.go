package storage

import (
	"testing"
	"time"
)

func TestQueueBoundsDropLowPriority(t *testing.T) {
	s, err := Open(t.TempDir(), 700, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Enqueue(QueueItem{ID: "low", Kind: "telemetry", Priority: Low, Payload: make([]byte, 600)}); err != nil {
		t.Fatal(err)
	}
	if err := s.Enqueue(QueueItem{ID: "high", Kind: "job-result", Priority: High, Payload: make([]byte, 600)}); err != nil {
		t.Fatal(err)
	}
	items, err := s.Dequeue(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "high" {
		t.Fatalf("unexpected items: %+v", items)
	}
}
