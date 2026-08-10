# Troubleshooting

* Enrollment fails with certificate errors: start `device-simulator` first so
  `.dev-control-plane/ca.pem` exists, then retry enrollment.
* Agent is offline: inspect `edge-agentctl queue status` for buffered telemetry
  and heartbeat items.
* Docker jobs fail: verify the Compose definition is marked managed and does not
  request privileged mode, host networking, arbitrary mounts, or Docker socket
  mounts.
* Modbus reads fail: verify endpoint reachability, unit ID, allowlisted register
  addresses, timeout, and rate-limit settings.
