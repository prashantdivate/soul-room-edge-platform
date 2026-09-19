# Run The Platform Locally

The supported local workflow is the same on Windows, Linux, macOS, and WSL. It
requires Docker Desktop, or Docker Engine with Docker Compose v2.

## First Start

Run all commands in this section from the repository root.

The environment file is optional for a localhost-only evaluation. If owner
values are omitted, a clean installation uses:

| Field | Local fallback |
| --- | --- |
| Email | <code>admin@soulroom.local</code> |
| Password | <code>change-me-local</code> |

Do not expose that fallback password to a LAN or the internet. For normal use,
create the local environment file:

~~~bash
# Linux, macOS, or WSL
cp cloud-infra/.env.example cloud-infra/.env
~~~

~~~powershell
# Windows PowerShell
Copy-Item -Path .\cloud-infra\.env.example -Destination .\cloud-infra\.env
~~~

Here, <code>Copy-Item</code> copies the supplied template from
<code>-Path</code> to the private file named by <code>-Destination</code>.
Edit <code>cloud-infra/.env</code> before starting. At minimum, choose the
company identity and first organization owner:

~~~dotenv
SOULROOM_ORGANIZATION_NAME=Your Company
SOULROOM_ORGANIZATION_SLUG=your-company
SOULROOM_ADMIN_EMAIL=owner@your-company.com
SOULROOM_ADMIN_PASSWORD=choose-a-unique-password-with-12-or-more-characters
~~~

The values above are examples. On a clean data volume, Soul Room creates the
owner from the exact values in your <code>.env</code>. On an existing volume,
the original account remains in effect and bootstrap values are not reapplied.

Build and start the complete stack:

~~~bash
# Linux, macOS, or WSL
./platform.sh build
~~~

~~~bat
:: Windows Command Prompt or PowerShell
platform.cmd build
~~~

Open [http://localhost:3080](http://localhost:3080) and sign in with the owner
configured before the first start.

## Later Starts

Use the repository launcher instead of remembering Compose arguments:

| Action | Linux, macOS, WSL | Windows |
| --- | --- | --- |
| Start existing images | <code>./platform.sh up</code> | <code>platform.cmd up</code> |
| Rebuild after source changes | <code>./platform.sh refresh</code> | <code>platform.cmd refresh</code> |
| Stop and retain data | <code>./platform.sh down</code> | <code>platform.cmd down</code> |
| Show service state | <code>./platform.sh status</code> | <code>platform.cmd status</code> |
| Follow logs | <code>./platform.sh logs</code> | <code>platform.cmd logs</code> |
| Validate Docker and Compose | <code>./platform.sh doctor</code> | <code>platform.cmd doctor</code> |

## Local Addresses

| Service | Default address |
| --- | --- |
| Soul Room console | <code>http://localhost:3080</code> |
| Control API | <code>http://localhost:8080</code> |
| Device gateway | <code>https://localhost:8443</code> |
| ShellHub portal | <code>http://localhost:8088</code> |
| ShellHub SSH gateway | <code>localhost:22222</code> |

ShellHub performs a separate one-time setup for its administrator and first
namespace. Those accounts are stored in the local ShellHub data volume, not in
an external ShellHub cloud account.

## Connect A Physical Device

The device cannot use <code>localhost</code> to reach the computer running Soul
Room. Before the platform generates its gateway certificate, set the address
used by devices:

~~~dotenv
SOULROOM_DEVICE_PUBLIC_HOST=192.168.1.20
SOULROOM_SHELLHUB_URL=http://192.168.1.20:8088
~~~

Use a stable LAN address or DNS name. Do not copy an address from a screenshot
or hardcode another installation's host.

Then:

1. Start or refresh the platform.
2. Open **Management > Enrollment**.
3. Generate a one-time token.
4. Download <code>soul-room-dev-ca.pem</code>.
5. Build the agent bundle for the device CPU.
6. Install it with endpoint <code>https://YOUR_HOST:8443</code>.
7. Run <code>edge-agentctl status</code> on the device.

The complete device commands are in the
[agent README](../../agent/README.md) and
[embedded Linux installation guide](../../agent/docs/EMBEDDED_LINUX_INSTALLATION.md).

## Initialize Remote Access

Open **Operations > Remote access**. The ShellHub setup and console are
embedded in that page and use the same browser host as Soul Room. Create the
ShellHub administrator and namespace, then follow the
[ShellHub integration guide](SHELLHUB_INTEGRATION.md) to connect a device.

The Soul Room and ShellHub agents are separate. Install the ShellHub agent only
on devices where interactive SSH is allowed.

## Persistence

Normal starts, rebuilds, refreshes, and <code>down</code> preserve:

- Soul Room users, devices, telemetry, jobs, events, and certificates
- ShellHub users, namespaces, accepted devices, database records, and signing
  keys

See [Backup and restore](BACKUP_RESTORE.md) before upgrades or destructive
maintenance.

To erase a disposable local installation and initialize it again from the
current <code>.env</code>:

~~~bash
docker compose -f cloud-infra/compose.yaml down -v
docker compose -f cloud-infra/compose.yaml up --build -d
~~~

This permanently removes local accounts and fleet data. It is not a password
reset procedure for a customer installation.

## Troubleshooting

~~~bash
./platform.sh doctor
./platform.sh status
./platform.sh logs
~~~

On Windows, use the same commands through <code>platform.cmd</code>. For an
agent that does not appear in the console, continue with
[agent troubleshooting](../../agent/docs/TROUBLESHOOTING.md).
