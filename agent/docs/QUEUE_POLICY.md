# Offline Queue Policy

The queue is bounded by bytes and age. High-priority records such as job results
and security/audit events are retained ahead of low-priority telemetry.

When pressure occurs:

1. Expired low and normal priority items are dropped first.
2. Remaining items are sorted by priority and creation time.
3. The oldest low-priority items are dropped until the byte budget is satisfied.
4. Queue pressure is visible through `edge-agentctl queue status`.

The current backend is a crash-safe JSONL store. A SQLite backend with versioned
migrations is planned behind the same storage interface.
