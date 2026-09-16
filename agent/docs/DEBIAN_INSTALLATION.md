# Debian Installation

For Ubuntu or Debian, build the `embedded-linux-amd64`,
`embedded-linux-arm64`, or `embedded-linux-armv7` bundle described in the
agent README. Copy the bundle and downloaded Soul Room CA to the device, then
run its installer:

```bash
sudo sh install.sh --endpoint https://SOUL_ROOM_HOST:8443 --ca ./ca.pem --token ONE_TIME_TOKEN --name DEVICE_NAME
```

The service runs in the system context and does not create a separate Linux
account. Device identity is persisted under `/var/lib/edge-agent/identity`.
