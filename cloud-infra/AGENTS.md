# Repository Guidance

This repository contains the server-side control plane for Soul Room
Management.

Security-sensitive code paths live in `internal/auth`, `internal/rbac`,
`internal/enrollment`, `internal/certificates`, `internal/devicegateway`,
`internal/artifacts`, `internal/jobs`, and `internal/tenancy`.

Keep defaults restrictive:

* no default production passwords
* no hardcoded production CA keys
* no shared device private keys
* no UI-only authorization
* no unrestricted remote shell or port forwarding
* no permanent public object URLs
* no cross-tenant repository methods

When changing behavior, update `docs/IMPLEMENTATION_STATUS.md` and add focused
tests for the affected package.
