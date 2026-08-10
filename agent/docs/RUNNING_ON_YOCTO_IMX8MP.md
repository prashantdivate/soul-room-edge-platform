# Running On Yocto i.MX8MP

Recommended source/project path on the device, if you copy the project tree for
debugging:

```text
/opt/soul-room-platform/agent
```

Runtime paths used by the service:

```text
/usr/bin/edge-agent
/usr/bin/edge-agentctl
/etc/edge-agent/config.yaml
/etc/edge-agent/ca.pem
/var/lib/edge-agent
/var/lib/edge-agent/identity
/run/edge-agent
```

Build the agent for ARM64:

```cmd
set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=0
go build -o build/edge-agent ./cmd/edge-agent
```

```bash
cd agent
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/edge-agent ./cmd/edge-agent
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/edge-agentctl ./cmd/edge-agentctl
```

Install on the target:

```bash
install -m 0755 edge-agent /usr/bin/edge-agent
install -m 0755 edge-agentctl /usr/bin/edge-agentctl
install -d -m 0750 /etc/edge-agent /var/lib/edge-agent /var/lib/edge-agent/identity /run/edge-agent
install -m 0640 config.yaml /etc/edge-agent/config.yaml
```

Use `packaging/systemd/edge-agent.service` as the systemd unit. Then:

```bash
systemctl daemon-reload
systemctl enable --now edge-agent
journalctl -u edge-agent -f
```

For Yocto integration, use `packaging/yocto/edge-agent_0.1.0.bb` as the
starting recipe and make `/var/lib/edge-agent` persistent across rootfs updates.
