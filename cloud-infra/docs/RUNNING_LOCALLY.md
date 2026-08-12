# Run the Platform Locally

The local workflow is the same on Windows, Linux, and macOS. It requires only
Docker Desktop or Docker Engine with the Compose plugin.

## Start

Open the VS Code terminal in `cloud-infra`, then run:

```text
docker compose up --build -d
```

Copy `.env.example` to `.env` and set the company and initial owner values:

```dotenv
SOULROOM_ORGANIZATION_NAME=Acme Devices
SOULROOM_ORGANIZATION_SLUG=acme-devices
SOULROOM_ADMIN_EMAIL=admin@acme.com
SOULROOM_ADMIN_PASSWORD=replace-with-a-long-unique-password
```

Open `http://localhost:3080` and sign in with that owner account. These values
are used only when the data volume is empty; normal restarts never reset the
owner password.

Compose starts the web console, control API, device gateway, worker, PostgreSQL,
MinIO, and Mailpit. On the first run it creates only the local administrator and
organization. Devices and fleet activity appear only after a real agent enrolls.

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

The command prints the backup-set timestamp. Restore that set with the same
command on Windows, Linux, or macOS:

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
| MinIO console | `http://localhost:9001` |
| Mailpit | `http://localhost:8025` |

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
```

Docker Compose reads this file on Windows, Linux, and macOS. It ensures the
gateway certificate matches the address used by the device.

## Development Note

The platform and seed containers run as root only to initialize the local named
volume. Production deployments should pre-create writable volumes and run the
service as a non-root user.
