# Remote Access Security Boundary

Interactive maintenance is delegated to the bundled self-hosted ShellHub
Community Edition stack. The Soul Room device protocol does not accept arbitrary
shell commands, and the fleet agent never opens an inbound device port.

Soul Room verifies tenant membership and `remote.session.create` permission,
requires a tenant-owned device with an explicitly mapped ShellHub SSHID, and
writes a `remote_access.requested` audit event before returning the local
ShellHub portal and SSH command.

ShellHub owns the outbound reverse tunnel, Linux authentication, public keys,
firewall rules, and device acceptance. Operators must use short sessions,
public-key authentication, least-privileged Linux users, and a reviewed firewall
policy. Soul Room does not store Linux passwords or operator private keys.

Shared SSO, automatic namespace/device mapping, approval workflows, and
short-lived session brokering remain deferred. Production deployments require
trusted HTTPS, WebSocket-aware proxying, monitored backups, rate limiting, and
an edition of ShellHub that provides any promised enterprise audit, recording,
MFA, or HA capability.

See `SHELLHUB_INTEGRATION.md` for installation and operation.
