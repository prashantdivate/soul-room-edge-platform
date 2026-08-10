# Remote Access

Unrestricted remote shell access is not implemented in this MVP.

Any future tunnel design must require unique device identity, short-lived
session credentials, per-session device and port authorization, no reusable
shared SSH keys, tunnel-only accounts, no interactive shell on the tunnel
server, no PTY, no agent forwarding, no X11 forwarding, session expiry,
complete audit logging, rate limiting, and optional approval workflows.
