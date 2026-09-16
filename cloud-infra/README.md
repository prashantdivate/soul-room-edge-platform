# Soul Room Control Plane

The Soul Room control plane provides the web console, tenant-aware API, device
gateway, durable local state, and bundled ShellHub remote-access services for
Linux edge fleets.

Use the repository [README](../README.md) for the shortest end-to-end workflow,
including building and enrolling the edge agent.

## Start Locally

From the repository root, create <code>cloud-infra/.env</code> from
<code>cloud-infra/.env.example</code>. Choose your own organization name,
owner email, and owner password before the first start.

~~~bash
# Linux, macOS, or WSL
cp cloud-infra/.env.example cloud-infra/.env
./platform.sh build
~~~

~~~bat
:: Windows Command Prompt or PowerShell
copy cloud-infra\.env.example cloud-infra\.env
platform.cmd build
~~~

Open [http://localhost:3080](http://localhost:3080).

There is no shared login from this README. A new installation creates its first
owner from <code>SOULROOM_ADMIN_EMAIL</code> and
<code>SOULROOM_ADMIN_PASSWORD</code> in <code>.env</code>. An existing data
volume keeps the account created during its first initialization; changing
<code>.env</code> does not overwrite that account.

The local launcher also supports <code>up</code>, <code>down</code>,
<code>refresh</code>, <code>restart</code>, <code>status</code>,
<code>logs</code>, and <code>doctor</code>. See
[Run the platform locally](docs/RUNNING_LOCALLY.md) for networking, service
addresses, persistence, and first-device setup.

## Current Capabilities

| Area | Implemented |
| --- | --- |
| Fleet | Device enrollment, certificate identity, presence, telemetry, inventory, gateways, and map locations |
| Operations | Typed jobs, offline queueing, diagnostics, deployments, update campaigns, alerts, and exports |
| Administration | Tenant-scoped users and roles, settings, audit history, quotas, backup, and restore |
| Remote access | Self-hosted ShellHub Community Edition embedded in the Soul Room console |
| Updates | Capability-gated Mender, RAUC, OSTree, SWUpdate, Flatpak, and custom-adapter campaigns |
| Software posture | Debian/RPM inventory and optional on-demand Trivy advisory results |

The default Compose stack uses a local durable Soul Room store plus ShellHub's
own PostgreSQL and Valkey services. Unused Soul Room PostgreSQL, MinIO, and
Mailpit containers are not started.

## Configuration

Configuration is intentionally split by responsibility:

- <code>.env</code> holds bootstrap identity, network endpoints, secrets, and
  infrastructure values.
- **Settings > Platform settings** holds safe organization-level operational
  defaults that owners may change without restarting containers.
- Docker volumes hold runtime state; source control contains no registered
  devices, telemetry, credentials, or generated certificates.

See [On-premises administration](docs/ONPREM_ADMIN.md) for the complete
boundary.

## Documentation

| Task | Guide |
| --- | --- |
| Launch and connect a physical device | [Run locally](docs/RUNNING_LOCALLY.md) |
| Configure the deployment | [On-premises administration](docs/ONPREM_ADMIN.md) |
| Configure remote SSH | [ShellHub integration](docs/SHELLHUB_INTEGRATION.md) |
| Back up or restore state | [Backup and restore](docs/BACKUP_RESTORE.md) |
| Understand the system | [System architecture](docs/SYSTEM_ARCHITECTURE.md) |
| Review stored records | [Data model](docs/DATA_MODEL.md) |
| Review security | [Threat model](docs/THREAT_MODEL.md) |
| Check implemented and deferred work | [Implementation status](docs/IMPLEMENTATION_STATUS.md) |

## Development Check

From the repository root:

~~~bash
docker build -f cloud-infra/deploy/compose/service.Dockerfile --target test cloud-infra
docker compose -f cloud-infra/compose.yaml config
~~~

Soul Room is an implemented MVP foundation. Review the implementation status
and threat model before treating it as production-ready infrastructure.
