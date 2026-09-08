# Run the Platform Locally

The local workflow is the same on Windows, Linux, and macOS. It requires only
Docker Desktop or Docker Engine with the Compose plugin.

## Start

Open the VS Code terminal at the repository root, then run:

```text
./platform.sh build
# Windows Command Prompt or PowerShell: platform.cmd build
```

After the images exist, use `./platform.sh up` or `platform.cmd up` for a fast
start without rebuilding.

Copy `.env.example` to `.env` and set the company and initial owner values:

```dotenv
SOULROOM_ORGANIZATION_NAME=Acme Devices
SOULROOM_ORGANIZATION_SLUG=acme-devices
SOULROOM_ADMIN_EMAIL=admin@acme.com
SOULROOM_ADMIN_PASSWORD=replace-with-a-long-unique-password
SOULROOM_SHELLHUB_DOMAIN=localhost
SOULROOM_SHELLHUB_URL=http://localhost:8088
SOULROOM_SHELLHUB_POSTGRES_PASSWORD=replace-with-a-long-unique-password
```

Open `http://localhost:3080` and sign in with that owner account. These values
are used only when the data volume is empty; normal restarts never reset the
owner password.

Compose starts the web console, combined control-plane service, and the pinned
ShellHub Community Edition stack. Soul Room's current durable store lives in
the `app-data` volume, so unused PostgreSQL, MinIO, and Mailpit containers are
not part of the default local stack. On the first run it creates only the Soul
Room administrator and organization. ShellHub uses
its own security boundary and asks you to create its administrator and namespace
at `http://localhost:8088/setup`. Devices and fleet activity appear only after
a real agent enrolls.

## Stop

```text
docker compose down
```

Your local data remains in Docker volumes. To view service output:

```text
docker compose logs -f
```

## Backup and Restore

Create a timestamped local backup set in `cloud-infra/backups`:

```text
docker compose --profile maintenance run --rm backup
```

The backup includes Soul Room state, ShellHub PostgreSQL records, and ShellHub
signing keys. The command prints the backup-set timestamp. Restore
that set with the same command on Windows, Linux, or macOS:

```text
docker compose --profile maintenance run --rm -e BACKUP_SET=YYYYmmddHHMMSS restore
```

Use a maintenance window for restore and validate login, enrollment, telemetry,
and audit history afterward.

## Local Addresses

| Service | Address |
| --- | --- |
| Web console | `http://localhost:3080` |
| Control API | `http://localhost:8080` |
| Device gateway | `https://localhost:8443` |
| ShellHub portal | `http://localhost:8088` |
| ShellHub SSH gateway | `localhost:22222` |

## Connect a Physical Device

1. Open **Enrollment** in the web console.
2. Generate a one-time token.
3. Download the development CA from the same screen.
4. Install the CA as `/etc/edge-agent/ca.pem` on the device.
5. Set the agent endpoint to `https://<computer-ip>:8443`.

Before starting Compose for a physical device, copy `.env.example` to `.env`
and replace `localhost` with the IP address or DNS name the device uses to reach
this computer:

```text
SOULROOM_DEVICE_PUBLIC_HOST=192.168.1.20
SOULROOM_SHELLHUB_DOMAIN=localhost
SOULROOM_SHELLHUB_URL=http://192.168.1.20:8088
```

Docker Compose reads this file on Windows, Linux, and macOS. It ensures the
gateway certificate matches the address used by the device.

## Initialize Remote Access

1. Open **Remote access** and use the embedded portal, or open `http://localhost:8088/setup` directly, then create the ShellHub administrator.
2. Create a ShellHub namespace and copy its tenant ID.
3. Follow `SHELLHUB_INTEGRATION.md` to install the ShellHub agent on the device.
4. Accept the pending device in ShellHub and copy its SSHID.
5. Save that SSHID against the same device on Soul Room's **Remote access** page.

ShellHub keys and database records live in the `shellhub-keys` and
`shellhub-postgres-data` named volumes. Normal rebuilds and `docker compose down`
preserve them. `docker compose down -v` permanently removes both Soul Room and
ShellHub local data.

## Development Note

The platform and seed containers run as root only to initialize the local named
volume. Production deployments should pre-create writable volumes and run the
service as a non-root user.
