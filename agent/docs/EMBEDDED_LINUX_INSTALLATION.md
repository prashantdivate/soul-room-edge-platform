# Embedded Linux Installation

The agent is generic Linux software. The same binaries and service layout work
on Ubuntu, Debian, and compatible embedded distributions; choose the bundle
that matches the device CPU.

## Build On Any Host

From the `agent` directory, export a bundle with Docker:

```bash
# x86-64: amd64 / x86_64
docker build --target embedded-linux-amd64 --output type=local,dest=dist/embedded-linux-amd64 .

# 64-bit ARM: aarch64 / arm64
docker build --target embedded-linux-arm64 --output type=local,dest=dist/embedded-linux-arm64 .

# 32-bit ARM: armv7 / armhf
docker build --target embedded-linux-armv7 --output type=local,dest=dist/embedded-linux-armv7 .
```

Each bundle contains `edge-agent`, `edge-agentctl`, `config.yaml`,
`edge-agent.service`, and `install.sh`. Go is not required on the build host.

## Install On The Device

Use `/opt/soul-room-agent` as a temporary staging directory. Copy the matching
bundle and the CA downloaded from the web console into it. Create a one-time
token under **Enrollment**, then run:

```bash
cd /opt/soul-room-agent
sudo sh install.sh \
  --endpoint https://SOUL_ROOM_HOST:8443 \
  --ca ./soul-room-dev-ca.pem \
  --token YOUR_TOKEN \
  --name DEVICE_NAME
```

The service runs in the system service context. It does not create or require a
dedicated Linux user. On a systemd device, the installer installs and enables
the bundled unit. For another init system, integrate the same binaries and
configuration with that image's service manager; see
[Yocto integration](YOCTO_INTEGRATION.md).

## Enroll And Start

The installer enrolls and starts the service. Verify it at any time with:

```bash
sudo edge-agentctl -config /etc/edge-agent/config.yaml status
sudo systemctl status edge-agent
sudo journalctl -u edge-agent -f
```

To change the endpoint, telemetry, location, OTA, or other agent settings later,
use the built-in terminal UI. It saves the validated configuration on exit:

```bash
sudo edge-agentctl -config /etc/edge-agent/config.yaml tui
sudo systemctl restart edge-agent
```

For an already enrolled device, preserve `/var/lib/edge-agent/identity` and
replace only the binaries and service file.

## OS And Hardware Inventory

The agent uses the running kernel for CPU architecture and kernel details. It
reads `/etc/os-release`, then `/usr/lib/os-release`, for distribution name,
version, and optional `BUILD_ID`. This works for both general-purpose Linux
and Yocto images. Fields that the image does not provide remain empty instead
of being replaced with Ubuntu-specific values.

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
  ip_url: https://ipwho.is/
```

For mobile hardware with `gpsd`, use `source: gpsd`. Operators can also set or
correct a fixed site from **Fleet map**. When GPS is unavailable, use
`source: ip` for an explicitly enabled, network-approximate location. This
contacts the configured HTTPS `ip_url`; use GPSD or static coordinates when
site-level accuracy or no third-party lookup is required.
