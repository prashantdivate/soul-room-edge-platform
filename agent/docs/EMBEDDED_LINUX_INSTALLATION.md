# Embedded Linux Installation

The agent is generic Linux software. The same binaries and service layout work
on Ubuntu, Debian, and compatible embedded distributions; choose the bundle
that matches the device CPU.

## Build On Any Host

From the `agent` directory, export a bundle with Docker:

```bash
# 64-bit ARM: aarch64 / arm64
docker build --target embedded-linux-arm64 --output type=local,dest=dist/embedded-linux-arm64 .

# 32-bit ARM: armv7 / armhf
docker build --target embedded-linux-armv7 --output type=local,dest=dist/embedded-linux-armv7 .
```

Each bundle contains `edge-agent`, `edge-agentctl`, `config.yaml`, and
`edge-agent.service`. Go is not required on the build host.

## Install On The Device

Use `/opt/soul-room-agent` as a temporary staging directory. Copy the matching
bundle and the CA downloaded from the web console into it, then run:

```bash
cd /opt/soul-room-agent
sudo install -m 0755 edge-agent edge-agentctl /usr/bin/
sudo install -d -m 0750 /etc/edge-agent /var/lib/edge-agent /var/lib/edge-agent/identity
sudo install -m 0644 config.yaml /etc/edge-agent/config.yaml
sudo install -m 0644 soul-room-dev-ca.pem /etc/edge-agent/ca.pem
sudo install -m 0644 edge-agent.service /etc/systemd/system/edge-agent.service
```

Edit `/etc/edge-agent/config.yaml` and set `server.endpoint` to the HTTPS
device-gateway address reachable from the device. Do not commit a private LAN
address to this repository.

The service runs in the system service context. It does not create or require a
dedicated Linux user.

## Enroll And Start

Create a token from **Enrollment** in the console. Flags for enrollment belong
after the `enroll` subcommand:

```bash
sudo edge-agentctl -config /etc/edge-agent/config.yaml enroll -token YOUR_TOKEN -name DEVICE_NAME
sudo systemctl daemon-reload
sudo systemctl enable --now edge-agent
sudo systemctl status edge-agent
sudo journalctl -u edge-agent -f
```

For an already enrolled device, preserve `/var/lib/edge-agent/identity` and
replace only the binaries and service file.

## Device Location

Location is disabled by default and is never guessed. For a fixed installation,
set the deployed configuration before restarting:

```yaml
location:
  source: static
  latitude: 18.5204
  longitude: 73.8567
  label: Pune lab
  gpsd_address: 127.0.0.1:2947
```

For mobile hardware with `gpsd`, use `source: gpsd`. Operators can also set or
correct a fixed site from **Fleet map**.
