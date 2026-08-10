# Architecture

The agent is a single Linux service with narrow internal subsystems:

* `identity`: local key generation, identity persistence, signer abstraction.
* `enrollment`: one-time token enrollment and development local CA support.
* `transport`: outbound HTTPS client, TLS validation, retry and replay metadata.
* `storage`: durable bounded queues and persisted job state.
* `telemetry`: collector scheduler and normalized metrics.
* `inventory`: normalized device and downstream inventory.
* `jobs`: typed job validation, execution, recovery, results, and idempotency.
* `applications`: safe file deployment and application state adapters.
* `containers`: managed container policy and provider abstraction.
* `gateway` and `connectors`: downstream device representation and connectors.
* `ota`: OTA adapter interface and simulator adapter.
* `observability` and `audit`: structured logs, counters, and audit events.

## Trust Boundary

```mermaid
flowchart LR
  Cloud["Control plane"] -->|TLS server auth + optional mTLS| Agent["edge-agent"]
  Agent --> Store["Local state"]
  Agent --> Docker["Docker API"]
  Agent --> Files["Allowlisted file destinations"]
  Agent --> Gateway["Connector framework"]
  Gateway --> Modbus["Read-only Modbus TCP"]
  Gateway --> Sim["Simulator devices"]
```

## Connection Lifecycle

```mermaid
sequenceDiagram
  participant A as Agent
  participant C as Control plane
  A->>A: Generate local private key
  A->>C: Enroll(token, public key, fingerprint)
  C-->>A: Device ID, tenant, certificate, trusted config
  A->>C: Hello(identity, capabilities)
  loop Online
    A->>C: Heartbeat, inventory, telemetry batches
    C-->>A: Typed jobs
    A->>C: Job progress and result
  end
```
