# Job State Machine

```mermaid
stateDiagram-v2
  [*] --> received
  received --> rejected: malformed/expired/unauthorized
  received --> prepared: validation passed
  prepared --> running
  running --> succeeded
  running --> failed
  running --> cancelled
  failed --> retry_pending: retryable
  retry_pending --> running
```

Jobs are typed and allowlisted. Each job must include job ID, tenant ID, device
ID, type, expiry, attempt, scopes, payload, and optional payload digest.

Completed successful jobs are persisted and are not silently rerun after
reconnect or restart.
