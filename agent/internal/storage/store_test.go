package storage

import (
	"bytes"
	"os"
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

func TestArchiveQueuePreservesRecordsAndClearsActiveQueue(t *testing.T) {
	s, err := Open(t.TempDir(), 1024*1024, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Enqueue(QueueItem{ID: "heartbeat-1", Kind: "heartbeat", Priority: High, Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	archive, err := s.ArchiveQueue()
	if err != nil {
		t.Fatal(err)
	}
	if archive == "" {
		t.Fatal("expected an archive path")
	}
	if _, err := os.Stat(archive); err != nil {
		t.Fatalf("archived queue is missing: %v", err)
	}
	stats, err := s.Stats()
	if err != nil || stats.Items != 0 {
		t.Fatalf("active queue was not cleared: %+v err=%v", stats, err)
	}
}

func TestQueueReadsInventoryLargerThanScannerDefault(t *testing.T) {
	s, err := Open(t.TempDir(), 1024*1024, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte("inventory-data-"), 16*1024)
	if err := s.Enqueue(QueueItem{ID: "inventory-1", Kind: "inventory", Priority: Normal, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	items, err := s.Dequeue(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !bytes.Equal(items[0].Payload, payload) {
		t.Fatalf("large inventory payload did not round trip: items=%d", len(items))
	}
}
