package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unified-fleet/edge-agent/internal/storage"
)

type Job struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	DeviceID       string          `json:"device_id"`
	Type           string          `json:"type"`
	CreatedAt      time.Time       `json:"created_at"`
	ExpiresAt      time.Time       `json:"expires_at"`
	Attempt        int             `json:"attempt"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	PayloadDigest  string          `json:"payload_digest"`
	Scopes         []string        `json:"scopes"`
	Payload        json.RawMessage `json:"payload"`
}

type Result struct {
	JobID      string    `json:"job_id"`
	State      string    `json:"state"`
	ExitCode   int       `json:"exit_code"`
	Stdout     string    `json:"stdout,omitempty"`
	Stderr     string    `json:"stderr,omitempty"`
	ErrorClass string    `json:"error_class,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Retryable  bool      `json:"retryable"`
}

type ProgressReporter interface {
	Progress(ctx context.Context, jobID, message string) error
}

type Handler interface {
	Validate(context.Context, Job) error
	Prepare(context.Context, Job) error
	Execute(context.Context, Job, ProgressReporter) Result
	Cancel(context.Context, string) error
	Recover(context.Context, storage.JobState) Result
}

type Registry struct {
	store    *storage.Store
	handlers map[string]Handler
}

func NewRegistry(store *storage.Store) *Registry {
	return &Registry{store: store, handlers: map[string]Handler{}}
}

func (r *Registry) Register(jobType string, h Handler) {
	r.handlers[jobType] = h
}

func (r *Registry) Execute(ctx context.Context, job Job, reporter ProgressReporter) Result {
	start := time.Now()
	if err := job.Validate(time.Now()); err != nil {
		return fail(job.ID, start, "validation", err)
	}
	if st, ok, err := r.store.LoadJobState(job.ID); err == nil && ok && st.State == "succeeded" {
		var prev Result
		_ = json.Unmarshal(st.ResultJSON, &prev)
		return prev
	}
	h, ok := r.handlers[job.Type]
	if !ok {
		return fail(job.ID, start, "unsupported_type", fmt.Errorf("unsupported job type %s", job.Type))
	}
	if err := h.Validate(ctx, job); err != nil {
		return fail(job.ID, start, "validation", err)
	}
	if err := h.Prepare(ctx, job); err != nil {
		return fail(job.ID, start, "prepare", err)
	}
	_ = r.store.SaveJobState(storage.JobState{JobID: job.ID, State: "running"})
	res := h.Execute(ctx, job, reporter)
	if res.JobID == "" {
		res.JobID = job.ID
	}
	if res.StartedAt.IsZero() {
		res.StartedAt = start
	}
	if res.FinishedAt.IsZero() {
		res.FinishedAt = time.Now()
	}
	if len(res.Stdout) > 64*1024 {
		res.Stdout = res.Stdout[:64*1024]
	}
	if len(res.Stderr) > 64*1024 {
		res.Stderr = res.Stderr[:64*1024]
	}
	raw, _ := json.Marshal(res)
	_ = r.store.SaveJobState(storage.JobState{JobID: job.ID, State: res.State, ResultJSON: raw})
	return res
}

func (j Job) Validate(now time.Time) error {
	if j.ID == "" || j.TenantID == "" || j.DeviceID == "" || j.Type == "" {
		return errors.New("job missing required identity fields")
	}
	if !j.ExpiresAt.IsZero() && now.After(j.ExpiresAt) {
		return errors.New("job expired")
	}
	if j.Attempt < 0 {
		return errors.New("invalid attempt")
	}
	if j.PayloadDigest != "" {
		sum := sha256.Sum256(j.Payload)
		if hex.EncodeToString(sum[:]) != strings.ToLower(j.PayloadDigest) {
			return errors.New("payload digest mismatch")
		}
	}
	return nil
}

func fail(jobID string, started time.Time, class string, err error) Result {
	return Result{JobID: jobID, State: "failed", ExitCode: 1, Stderr: err.Error(), ErrorClass: class, StartedAt: started, FinishedAt: time.Now()}
}

type NoopReporter struct{}

func (NoopReporter) Progress(context.Context, string, string) error { return nil }

type FunctionHandler struct {
	validate func(context.Context, Job) error
	execute  func(context.Context, Job, ProgressReporter) Result
}

func NewFunctionHandler(validate func(context.Context, Job) error, execute func(context.Context, Job, ProgressReporter) Result) FunctionHandler {
	return FunctionHandler{validate: validate, execute: execute}
}

func (h FunctionHandler) Validate(ctx context.Context, j Job) error {
	if h.validate != nil {
		return h.validate(ctx, j)
	}
	return nil
}

func (h FunctionHandler) Prepare(context.Context, Job) error { return nil }

func (h FunctionHandler) Execute(ctx context.Context, j Job, r ProgressReporter) Result {
	if h.execute != nil {
		return h.execute(ctx, j, r)
	}
	return Result{JobID: j.ID, State: "succeeded", StartedAt: time.Now(), FinishedAt: time.Now()}
}

func (h FunctionHandler) Cancel(context.Context, string) error { return nil }

func (h FunctionHandler) Recover(context.Context, storage.JobState) Result {
	return Result{State: "failed", ErrorClass: "recover_unimplemented"}
}
