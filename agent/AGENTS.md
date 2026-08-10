# Repository Guidance

This repository contains the Linux edge-device agent for Unified Fleet Management.

Security-sensitive code paths live in `internal/identity`, `internal/enrollment`,
`internal/transport`, `internal/jobs`, `internal/applications`, and
`internal/connectors`. Keep defaults restrictive: no shared fleet keys, no TLS
verification bypasses, no unrestricted shell, no arbitrary downstream writes, and
no unconstrained file deployment.

When changing behavior, update `docs/IMPLEMENTATION_STATUS.md` and add focused
tests for the affected package.
