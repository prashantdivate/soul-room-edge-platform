# Raspberry Pi Installation

The Raspberry Pi must be on the same network as the platform computer. The
current device-gateway address is `https://192.168.29.146:8443`.

## Build On Any Host

From the `agent` directory, export the ARM64 bundle with Docker:

```text
docker build --target raspberry-pi --output type=local,dest=dist/raspberry-pi .
```

This creates `edge-agent`, `edge-agentctl`, `config.yaml`, and
`edge-agent.service` without requiring Go on the host computer.

## Install On Ubuntu

Use `/opt/unified-fleet-agent` as the project staging directory on the device.
Copy `dist/raspberry-pi` and the CA downloaded from the web console into that
directory. On the Pi:

```text
cd /opt/unified-fleet-agent
sudo install -m 0755 edge-agent edge-agentctl /usr/bin/
sudo install -d -m 0750 /etc/edge-agent /var/lib/edge-agent /var/lib/edge-agent/identity
sudo install -m 0644 config.yaml /etc/edge-agent/config.yaml
sudo install -m 0644 unified-fleet-dev-ca.pem /etc/edge-agent/ca.pem
sudo install -m 0644 edge-agent.service /etc/systemd/system/edge-agent.service
```

The installed runtime paths are `/usr/bin/edge-agent`,
`/etc/edge-agent/config.yaml`, and `/var/lib/edge-agent`. The staging directory
can be removed after installation. The systemd unit uses the system service
context and does not create or require a separate Linux user.

## Enroll And Start

Create a token from **Enrollment** in the web console, then run:

```text
sudo edge-agentctl -config /etc/edge-agent/config.yaml enroll -token YOUR_TOKEN -name raspberry-pi
sudo systemctl daemon-reload
sudo systemctl enable --now edge-agent
```

For an already enrolled device, preserve `/var/lib/edge-agent/identity` and
replace only the binaries and service file:

```text
sudo install -m 0755 edge-agent edge-agentctl /usr/bin/
sudo install -m 0644 edge-agent.service /etc/systemd/system/edge-agent.service
sudo systemctl daemon-reload
sudo systemctl restart edge-agent
```

Check the first live report:

```text
sudo systemctl status edge-agent
sudo journalctl -u edge-agent -f
```

The device appears in the console after enrollment. CPU, memory, disk,
temperature, network, uptime, OS, kernel, hardware model, serial number, and
architecture update every 30 seconds.

## Device Location

Location is disabled by default and is never guessed. For a fixed installation,
set the deployed `/etc/edge-agent/config.yaml` values before restarting:

```text
location:
  source: static
  latitude: 18.5204
  longitude: 73.8567
  label: Pune lab
  gpsd_address: 127.0.0.1:2947
```

For a mobile device with GPS hardware and `gpsd`, use `source: gpsd`. The agent
accepts a 2D or 3D GPS fix and reports its accuracy. Operators can also set or
correct a fixed site from **Fleet map** in the web console.

If the platform computer's IP address changes, update both
`cloud-infra/.env` and `/etc/edge-agent/config.yaml`, restart Compose, then
restart the agent.
