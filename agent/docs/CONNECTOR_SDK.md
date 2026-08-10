# Connector SDK Guide

Connectors implement `pkg/connectorsdk.Connector` and default to read-only
monitoring. Each downstream device receives a stable ID, parent gateway ID,
connector type, protocol address, model, firmware version, health state,
capabilities, and command allowlist.

Commands must be connector-specific and allowlisted. The cloud is never part of
a real-time control loop.

## Simulator

`internal/connectors.SimulatorConnector` returns deterministic simulated
controller and meter devices for local development.

## Modbus TCP

`internal/connectors.ModbusTCPConnector` reads configured allowlisted holding
registers only. It enforces timeouts and rate limits and does not implement
arbitrary writes.
