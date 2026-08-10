# Debian Installation

Build binaries for the target architecture, install them under `/usr/bin`, copy
`packaging/systemd/edge-agent.service` to the systemd unit directory, and place
configuration at `/etc/edge-agent/config.yaml`.

Create an `edge-agent` system user with no shell, then create:

* `/var/lib/edge-agent`
* `/var/lib/edge-agent/identity`
* `/run/edge-agent`

Use the systemd unit hardening options shipped in `packaging/systemd`.
