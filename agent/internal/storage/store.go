package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Priority int

const (
	Low Priority = iota
	Normal
	High
)

type QueueItem struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Priority    Priority  `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	Payload     []byte    `json:"payload"`
	ApproxBytes int64     `json:"approx_bytes"`
}

type QueueStats struct {
	Items        int   `json:"items"`
	Bytes        int64 `json:"bytes"`
	DroppedItems int   `json:"dropped_items"`
}

type Store struct {
	dir      string
	path     string
	maxBytes int64
	maxAge   time.Duration
	mu       sync.Mutex
	dropped  int
}

func Open(dir string, maxBytes int64, maxAge time.Duration) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Store{dir: dir, path: filepath.Join(dir, "queue.jsonl"), maxBytes: maxBytes, maxAge: maxAge}, nil
}

func (s *Store) Enqueue(item QueueItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		return fmt.Errorf("queue item id is required")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	item.ApproxBytes = int64(len(item.Payload)) + 256
	items, err := s.loadLocked()
	if err != nil {
		return err
	}
	items = append(items, item)
	items = s.prune(items, time.Now())
	return s.writeLocked(items)
}

func (s *Store) Dequeue(limit int) ([]QueueItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	out := append([]QueueItem(nil), items[:limit]...)
	remaining := append([]QueueItem(nil), items[limit:]...)
	return out, s.writeLocked(remaining)
}

func (s *Store) Stats() (QueueStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return QueueStats{}, err
	}
	var bytes int64
	for _, it := range items {
		bytes += it.ApproxBytes
	}
	return QueueStats{Items: len(items), Bytes: bytes, DroppedItems: s.dropped}, nil
}

func (s *Store) ArchiveQueue() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	archivePath := filepath.Join(s.dir, "queue.archived-"+time.Now().UTC().Format("20060102T150405.000000000Z")+".jsonl")
	if err := os.Rename(s.path, archivePath); err != nil {
		return "", err
	}
	return archivePath, nil
}

func (s *Store) loadLocked() ([]QueueItem, error) {
	f, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []QueueItem
	sc := bufio.NewScanner(f)
	maxInt := int64(^uint(0) >> 1)
	maxLine := maxInt
	if s.maxBytes <= maxInt/2 {
		maxLine = s.maxBytes * 2 // JSON base64 encoding can expand a binary payload by one third.
	}
	sc.Buffer(make([]byte, 64*1024), int(maxLine))
	for sc.Scan() {
		var it QueueItem
		if err := json.Unmarshal(sc.Bytes(), &it); err != nil {
			s.dropped++
			continue
		}
		out = append(out, it)
	}
	return out, sc.Err()
}

func (s *Store) writeLocked(items []QueueItem) error {
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, it := range items {
		if err := enc.Encode(it); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) prune(items []QueueItem, now time.Time) []QueueItem {
	keep := items[:0]
	for _, it := range items {
		if s.maxAge > 0 && now.Sub(it.CreatedAt) > s.maxAge && it.Priority < High {
			s.dropped++
			continue
		}
		keep = append(keep, it)
	}
	items = keep
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
		return items[i].Priority > items[j].Priority
	})
	var total int64
	for _, it := range items {
		total += it.ApproxBytes
	}
	for total > s.maxBytes && len(items) > 0 {
		// A single high-priority message is more valuable than an empty queue.
		// It will be retried and removed after the connection recovers.
		if len(items) == 1 && items[0].Priority == High {
			break
		}
		idx := len(items) - 1
		total -= items[idx].ApproxBytes
		items = items[:idx]
		s.dropped++
	}
	return items
}

type JobState struct {
	JobID      string    `json:"job_id"`
	State      string    `json:"state"`
	ResultJSON []byte    `json:"result_json,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Store) SaveJobState(st JobState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, "jobs.json")
	m := map[string]JobState{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	st.UpdatedAt = time.Now()
	m[st.JobID] = st
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func (s *Store) LoadJobState(jobID string) (JobState, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, "jobs.json")
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return JobState{}, false, nil
	}
	if err != nil {
		return JobState{}, false, err
	}
	m := map[string]JobState{}
	if err := json.Unmarshal(b, &m); err != nil {
		return JobState{}, false, err
	}
	st, ok := m[jobID]
	return st, ok, nil
}
