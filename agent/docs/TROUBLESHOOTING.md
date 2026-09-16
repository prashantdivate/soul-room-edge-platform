# Troubleshooting

* Enrollment fails with certificate errors: download the current CA from Soul
  Room's **Enrollment** page, install it as `/etc/edge-agent/ca.pem`, and retry.
* Agent is offline: run
  `sudo edge-agentctl -config /etc/edge-agent/config.yaml status`. It checks TLS
  and confirms that the device ID still exists in the platform.
* `x509: certificate signed by unknown authority`: the device and platform no
  longer use the same CA. This commonly follows `docker compose down -v` or a
  deleted `app-data` volume. Install the current CA and enroll the device again.
  A normal `docker compose down` followed by `up` preserves the CA and devices.
* `unknown_device`: the local identity exists but its platform record does not.
  Generate a new one-time token and run the release-bundle installer again. It
  archives the old queue, keeps the private key, installs the current CA, and
  enrolls the device again.
* Docker jobs fail: verify the Compose definition is marked managed and does not
  request privileged mode, host networking, arbitrary mounts, or Docker socket
  mounts.
* Modbus reads fail: verify endpoint reachability, unit ID, allowlisted register
  addresses, timeout, and rate-limit settings.
