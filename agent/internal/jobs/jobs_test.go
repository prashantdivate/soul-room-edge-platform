package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/soul-room/edge-agent/internal/storage"
)

func TestJobExpiryAndIdempotency(t *testing.T) {
	store, err := storage.Open(t.TempDir(), 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry(store)
	count := 0
	reg.Register("noop", NewFunctionHandler(nil, func(ctx context.Context, j Job, r ProgressReporter) Result {
		count++
		return Result{JobID: j.ID, State: "succeeded"}
	}))
	payload := json.RawMessage(`{"ok":true}`)
	sum := sha256.Sum256(payload)
	job := Job{ID: "j1", TenantID: "t1", DeviceID: "d1", Type: "noop", ExpiresAt: time.Now().Add(time.Hour), Payload: payload, PayloadDigest: hex.EncodeToString(sum[:])}
	res := reg.Execute(context.Background(), job, NoopReporter{})
	if res.State != "succeeded" {
		t.Fatalf("state = %s", res.State)
	}
	res = reg.Execute(context.Background(), job, NoopReporter{})
	if count != 1 || res.State != "succeeded" {
		t.Fatalf("idempotency failed count=%d state=%s", count, res.State)
	}
	job.ID = "j2"
	job.ExpiresAt = time.Now().Add(-time.Second)
	if reg.Execute(context.Background(), job, NoopReporter{}).ErrorClass != "validation" {
		t.Fatal("expected expiry validation")
	}
}
