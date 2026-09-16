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

For a quick device test, build the same ARM64 release bundle used by Ubuntu:

```bash
docker build --target embedded-linux-arm64 --output type=local,dest=dist/embedded-linux-arm64 agent
```

Copy that bundle and the CA downloaded from **Enrollment** to the target, then
use the same installer as every other systemd-based Linux device:

```bash
sudo sh install.sh --endpoint https://SOUL_ROOM_HOST:8443 --ca ./ca.pem --token ONE_TIME_TOKEN --name DEVICE_NAME
```

Then verify:

```bash
sudo edge-agentctl -config /etc/edge-agent/config.yaml status
```

For Yocto integration, use `packaging/yocto/edge-agent_0.1.0.bb` as the
starting recipe and make `/var/lib/edge-agent` persistent across rootfs updates.
