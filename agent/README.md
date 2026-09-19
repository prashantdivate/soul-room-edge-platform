# Soul Room Edge Agent

The Soul Room agent connects a Linux device to the Soul Room control plane. It
collects real inventory and telemetry, maintains device identity, buffers data
while offline, receives validated jobs, and drives only the update mechanisms
that are installed and configured on that device.

It is distribution-neutral Linux software. The same agent code runs on Ubuntu,
Debian, Raspberry Pi OS, and Yocto-based images; choose the binary for the
device CPU and integrate it with the device's service manager.

## Before You Start

You need:

- a running Soul Room platform reachable from the device
- a one-time token from **Management > Enrollment**
- <code>soul-room-dev-ca.pem</code> downloaded from the same page
- outbound device access to the Soul Room gateway, normally TCP port 8443
- root access for the packaged installer

Choose either Go 1.22 or Docker on the build computer. Neither tool is required
on a device that receives a prebuilt bundle.

## Build The Agent

Run <code>uname -m</code> on the target device:

| Device output | Docker target | Typical devices |
| --- | --- | --- |
| <code>x86_64</code> | <code>embedded-linux-amd64</code> | x86-64 PC, VM, industrial PC |
| <code>aarch64</code> or <code>arm64</code> | <code>embedded-linux-arm64</code> | 64-bit Raspberry Pi, i.MX8MP |
| <code>armv7l</code> or <code>armv7</code> | <code>embedded-linux-armv7</code> | 32-bit ARMv7 boards |

### Option 1: Build directly with Go

This is the shortest path when Go 1.22 or newer is installed on the Linux
device, or on a Linux build computer with the same CPU architecture:

~~~bash
cd agent
go test ./...
mkdir -p dist/native
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
  -o dist/native/edge-agent ./cmd/edge-agent
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
  -o dist/native/edge-agentctl ./cmd/edge-agentctl

cp configs/embedded-linux.yaml dist/native/config.yaml
cp packaging/systemd/edge-agent.service dist/native/
cp packaging/install.sh dist/native/
chmod +x dist/native/install.sh
~~~

The resulting <code>dist/native</code> directory is the same kind of
installation bundle used below. When building on a different CPU, cross-compile
both Go commands with these environment values:

| Target device | Go environment |
| --- | --- |
| x86-64 | <code>GOOS=linux GOARCH=amd64 CGO_ENABLED=0</code> |
| 64-bit ARM | <code>GOOS=linux GOARCH=arm64 CGO_ENABLED=0</code> |
| 32-bit ARMv7 | <code>GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0</code> |

For example, prefix each Go build command with
<code>GOOS=linux GOARCH=arm64 CGO_ENABLED=0</code> for an ARM64 target.
Environment-variable syntax differs in PowerShell, which is why Docker is the
simpler reproducible cross-build path on Windows.

### Option 2: Build reproducible bundles with Docker

From the <code>agent</code> directory, run the tests and export the bundle you
need:

~~~bash
docker build --target test .

# x86-64
docker build --target embedded-linux-amd64 \
  --output type=local,dest=dist/embedded-linux-amd64 .

# 64-bit ARM
docker build --target embedded-linux-arm64 \
  --output type=local,dest=dist/embedded-linux-arm64 .

# 32-bit ARMv7
docker build --target embedded-linux-armv7 \
  --output type=local,dest=dist/embedded-linux-armv7 .
~~~

To export every supported architecture on Linux or WSL, run
<code>make bundle</code>. Each output directory is self-contained:

~~~text
edge-agent
edge-agentctl
config.yaml
edge-agent.service
install.sh
~~~

The binaries are statically compiled with <code>CGO_ENABLED=0</code>; Go is not
required on the target.

## Deploy And Enroll

Copy the matching bundle and <code>soul-room-dev-ca.pem</code> to a temporary
directory on the device. <code>/opt/soul-room-agent</code> is a convenient
staging path:

~~~bash
cd /opt/soul-room-agent
sudo sh install.sh \
  --endpoint https://SOUL_ROOM_HOST:8443 \
  --ca ./soul-room-dev-ca.pem \
  --token ONE_TIME_TOKEN \
  --name DEVICE_NAME
~~~

Replace <code>SOUL_ROOM_HOST</code> with the DNS name or IP address shown on
the Enrollment page. It must be reachable from the device and covered by the
downloaded CA certificate. The token can be used only once.

The installer:

1. validates the bundle and HTTPS endpoint
2. installs the binaries, CA, configuration, and systemd unit
3. preserves an existing identity under <code>/var/lib/edge-agent/identity</code>
4. enrolls the device when a token is supplied
5. enables and starts <code>edge-agent</code> on a systemd device

It does not create a separate Linux user. The hardened service runs in the
system service context. On a non-systemd image, use the same binaries and
configuration but add the process to that image's init system. Yocto users can
start with the supplied BitBake recipe.

## Verify The Device

~~~bash
sudo edge-agentctl -config /etc/edge-agent/config.yaml status
sudo systemctl status edge-agent
sudo journalctl -u edge-agent -f
~~~

<code>status</code> checks the saved identity, local queue, TLS connection, and
whether the device still exists in Soul Room. A successful result reports
<code>"status": "connected"</code>. The device then appears under
**Fleet > Devices** after its first heartbeat.

Useful checks:

~~~bash
sudo edge-agentctl -config /etc/edge-agent/config.yaml config validate
sudo edge-agentctl -config /etc/edge-agent/config.yaml telemetry collect
sudo edge-agentctl -config /etc/edge-agent/config.yaml queue status
sudo edge-agentctl -config /etc/edge-agent/config.yaml identity show
sudo edge-agentctl -config /etc/edge-agent/config.yaml diagnostics create
sudo edge-agentctl version
~~~

<code>identity show</code> redacts certificates. Do not copy or commit
<code>/var/lib/edge-agent/identity</code>; it contains device-specific private
identity.

## Configure With The TUI

Open the terminal configuration editor:

~~~bash
sudo edge-agentctl -config /etc/edge-agent/config.yaml tui
~~~

The TUI edits six sections:

| Section | Settings |
| --- | --- |
| Server and trust | Gateway endpoint, CA path, connection timeout |
| Identity and storage | Identity path, TPM flag, offline queue size and age |
| Telemetry | Reporting interval, jitter, and individual collectors |
| Jobs and containers | Concurrency, timeout, shell policy, container provider |
| Gateway and location | Downstream connectors and <code>disabled</code>, <code>static</code>, <code>gpsd</code>, or <code>ip</code> location |
| OTA updates | Product compatibility, staging, trusted keys, limits, health check, reboot policy |

Press Enter to keep the displayed value. Enter <code>-</code> to clear an
optional text value. Durations use forms such as <code>30s</code>,
<code>10m</code>, or <code>2h</code>.

Choose **0 Save and exit** when finished. The TUI validates the complete
configuration and atomically writes <code>/etc/edge-agent/config.yaml</code>.
It does not restart a running service automatically, so apply the saved values
with:

~~~bash
sudo systemctl restart edge-agent
sudo edge-agentctl -config /etc/edge-agent/config.yaml status
~~~

Location is disabled by default. For a fixed device, select
<code>static</code> and enter latitude, longitude, and a label. Use
<code>gpsd</code> for mobile hardware with GPSD, or <code>ip</code> only when
an approximate third-party network lookup is acceptable.

## Runtime Paths

| Path | Purpose |
| --- | --- |
| <code>/usr/bin/edge-agent</code> | Long-running agent |
| <code>/usr/bin/edge-agentctl</code> | Enrollment, status, diagnostics, and TUI |
| <code>/etc/edge-agent/config.yaml</code> | Device configuration |
| <code>/etc/edge-agent/ca.pem</code> | Soul Room gateway CA |
| <code>/var/lib/edge-agent/identity</code> | Device key, certificate, and enrollment identity |
| <code>/var/lib/edge-agent</code> | Queue, OTA state, and diagnostics |
| <code>/usr/libexec/edge-agent/ota</code> | Optional custom OTA adapters |

The agent reads distribution details from <code>/etc/os-release</code>,
falling back to <code>/usr/lib/os-release</code>, and reads architecture and
kernel information from the running system. Ubuntu and Yocto therefore follow
the same reporting protocol; Yocto-specific <code>BUILD_ID</code> values are
reported when the image provides them.

## Updates And Optional Features

The agent advertises an OTA capability only when its updater is installed.
Mender, RAUC, OSTree, SWUpdate, and Flatpak still require correct device-side
storage, bootloader, signing trust, health confirmation, and rollback setup.
See [OTA adapters](docs/OTA_ADAPTERS.md) before enabling production updates.

Installed Debian and RPM packages are reported as inventory. Installing Trivy
adds the <code>security:trivy</code> capability and allows an operator to
request an on-demand OS-package advisory scan.

Remote SSH is intentionally separate. The agent never accepts arbitrary shell
commands; install the ShellHub agent only on devices where audited interactive
maintenance is allowed.

## Upgrade An Existing Agent

Run the matching new bundle's <code>install.sh</code> without a token to
replace the binaries and service while preserving the existing configuration
and identity:

~~~bash
sudo sh install.sh \
  --endpoint https://SOUL_ROOM_HOST:8443 \
  --ca ./soul-room-dev-ca.pem
~~~

Verify <code>status</code> after every upgrade. Preserve
<code>/var/lib/edge-agent</code> across read-only root filesystem updates and
A/B OS deployments.

## Documentation

| Task | Guide |
| --- | --- |
| Full Linux installation details | [Embedded Linux installation](docs/EMBEDDED_LINUX_INSTALLATION.md) |
| Ubuntu and Debian | [Debian installation](docs/DEBIAN_INSTALLATION.md) |
| Yocto image integration | [Yocto integration](docs/YOCTO_INTEGRATION.md) |
| i.MX8MP example | [Running on Yocto i.MX8MP](docs/RUNNING_ON_YOCTO_IMX8MP.md) |
| OTA prerequisites and custom adapters | [OTA adapters](docs/OTA_ADAPTERS.md) |
| Enrollment and connection failures | [Troubleshooting](docs/TROUBLESHOOTING.md) |
| Agent design | [Architecture](docs/ARCHITECTURE.md) |
| Security assumptions | [Threat model](docs/THREAT_MODEL.md) |
| Implemented and deferred work | [Implementation status](docs/IMPLEMENTATION_STATUS.md) |

## Security Defaults

- Device keys are generated locally and never sent to the server.
- TLS server certificates are validated.
- Jobs are typed, scoped, time-limited, and persisted for idempotency.
- Arbitrary remote shell is not part of the fleet protocol.
- File deployment uses destination allowlists, size limits, checksums, staging,
  and atomic rename where possible.
- Downstream connectors are read-only by default.

Soul Room is an implemented MVP foundation, not a claim of audited commercial
production readiness. Qualify update, rollback, power-loss, network-loss, and
recovery behavior on each supported hardware and image combination.
