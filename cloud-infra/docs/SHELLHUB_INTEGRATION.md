# ShellHub Integration

Unified Fleet delegates interactive SSH to ShellHub. This keeps SSH keys,
firewall rules, terminal transport, and session records out of the fleet API.
The integration does not expose port 22 on managed devices; the ShellHub agent
connects outbound to its server.

## Server

Deploy ShellHub Cloud or the official self-hosted ShellHub stack. Self-hosted
ShellHub has its own key generation and first-user setup, so it remains a
separate security boundary instead of being silently initialized with default
credentials by this Compose project.

Set the web endpoint in `cloud-infra/.env`:

```text
UFM_SHELLHUB_URL=https://shellhub.example.com
```

Recreate the platform service with `docker compose up -d platform`.

## Device

Install the ShellHub agent using the method supported by the target: container,
native Go binary, or the `meta-shellhub` Yocto layer. Accept the pending device
in ShellHub and copy its SSHID.

In Unified Fleet, open **Remote access**, select the device, and save that SSHID.
Every web-terminal or SSH-command request is then authorized by Unified Fleet
RBAC and written to its audit log before the operator is handed to ShellHub.

For production, require HTTPS, public-key authentication, ShellHub firewall
rules, short operator sessions, and session recording. Do not store Linux
passwords or reusable private keys in Unified Fleet.
