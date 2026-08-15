# Soul Room Edge Agent

Open-source Linux edge-device management agent for gateways, industrial PCs,
embedded Linux devices, and downstream controller fleets.

The agent supports secure local identity creation, development enrollment,
validated outbound HTTPS, heartbeat and telemetry reporting, bounded durable
offline buffering, typed jobs, safe file deployment, Docker policy validation,
Flatpak application updates, OTA adapter interfaces, and gateway connectors for simulated and read-only
Modbus TCP devices.

## Quick Start

```text
docker build --target test .
docker build --target embedded-linux-arm64 --output type=local,dest=dist/embedded-linux-arm64 .
docker build --target embedded-linux-armv7 --output type=local,dest=dist/embedded-linux-armv7 .
```

See `docs/EMBEDDED_LINUX_INSTALLATION.md` for installation and enrollment on
64-bit or 32-bit ARM devices.

## Optional Flatpak Updates

Install Flatpak on the device. A campaign can use an existing system remote or
an HTTPS `.flatpakrepo` descriptor from a self-hosted repository. The next
inventory report adds the `ota:flatpak` capability automatically. Soul Room preserves
GPG verification and runs only validated `flatpak update --system` jobs;
arbitrary command text is rejected. The packaged systemd unit does not create
or require a dedicated Linux user.

## Optional Package Advisory Scans

The agent reports installed Debian or RPM packages. When Trivy is installed it
also reports `security:trivy`, enabling an operator to queue an OS-package
advisory scan from the Applications page. Scanner findings are returned as job
evidence and are not converted into an invented production-readiness score.

For a full overview, see:

* `docs/IMPLEMENTATION_PLAN.md`
* `docs/ARCHITECTURE.md`
* `docs/IMPLEMENTATION_STATUS.md`
* `docs/THREAT_MODEL.md`

## Security Defaults

* Device keys are generated locally and never sent to the server.
* TLS server certificates are validated.
* Production enrollment requires signed server responses.
* Shell execution is not part of the fleet protocol; optional remote access is
  isolated in the separately installed ShellHub agent.
* Jobs are typed, validated, scoped, time-limited, and persisted for idempotency.
* File deployment uses destination allowlists, size limits, checksums, staging,
  and atomic rename where possible.
* Downstream connectors are read-only by default.

## Current Scope

This repository is an implemented MVP foundation, not a claim of final
commercial production readiness. See `docs/IMPLEMENTATION_STATUS.md` for the
implemented/deferred matrix.
