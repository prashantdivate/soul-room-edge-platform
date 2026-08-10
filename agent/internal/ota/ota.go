package ota

import (
	"context"
	"fmt"
	"sync"
)

type Metadata struct {
	UpdateID       string   `json:"update_id"`
	Version        string   `json:"version"`
	Product        string   `json:"product"`
	Architecture   string   `json:"architecture"`
	Digest         string   `json:"digest"`
	Signature      string   `json:"signature"`
	CompatibleFrom []string `json:"compatible_from"`
}

type Status struct {
	UpdateID string `json:"update_id"`
	State    string `json:"state"`
	Version  string `json:"version"`
	Error    string `json:"error,omitempty"`
}

type Adapter interface {
	CheckCompatibility(context.Context, Metadata) error
	Stage(context.Context, Metadata, string) error
	Verify(context.Context, Metadata) error
	Install(context.Context, Metadata) error
	Activate(context.Context, Metadata) error
	RequestReboot(context.Context, Metadata) error
	ConfirmHealth(context.Context, Metadata) error
	Rollback(context.Context, Metadata) error
	CurrentVersion(context.Context) (string, error)
	Status(context.Context) (Status, error)
}

type Simulator struct {
	mu      sync.Mutex
	version string
	status  Status
}

func NewSimulator(version string) *Simulator {
	return &Simulator{version: version, status: Status{State: "idle", Version: version}}
}

func (s *Simulator) CheckCompatibility(ctx context.Context, m Metadata) error {
	if m.Signature == "" || m.Digest == "" {
		return fmt.Errorf("OTA metadata requires digest and signature")
	}
	return nil
}

func (s *Simulator) Stage(context.Context, Metadata, string) error { return s.set("staged", "") }
func (s *Simulator) Verify(context.Context, Metadata) error        { return s.set("verified", "") }
func (s *Simulator) Install(context.Context, Metadata) error       { return s.set("installed", "") }
func (s *Simulator) Activate(context.Context, Metadata) error      { return s.set("activated", "") }
func (s *Simulator) RequestReboot(context.Context, Metadata) error {
	return s.set("reboot_required", "")
}

func (s *Simulator) ConfirmHealth(_ context.Context, m Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.version = m.Version
	s.status = Status{UpdateID: m.UpdateID, State: "succeeded", Version: m.Version}
	return nil
}

func (s *Simulator) Rollback(context.Context, Metadata) error { return s.set("rolled_back", "") }

func (s *Simulator) CurrentVersion(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.version, nil
}

func (s *Simulator) Status(context.Context) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status, nil
}

func (s *Simulator) set(state, err string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.State = state
	s.status.Error = err
	return nil
}
