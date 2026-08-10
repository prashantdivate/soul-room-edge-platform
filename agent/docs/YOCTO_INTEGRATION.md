# Yocto Integration

Use `packaging/yocto/edge-agent_0.1.0.bb` as the starting recipe. Persist
`/var/lib/edge-agent` across rootfs upgrades and keep `/etc/edge-agent` writable
or provisioned by the image build.

For read-only root filesystems, configure a persistent writable data partition
and bind mount it to `/var/lib/edge-agent` before starting the service.
