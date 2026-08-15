# OTA Adapters

Soul Room uses one durable OTA transaction engine for every update mechanism.
The engine owns download limits and resume, SHA-256 and release-key
verification, compatibility checks, phase persistence, reboot recovery, health
confirmation, commit, and rollback. An adapter owns only the native interaction
with the updater installed in the image.

## Built-In Adapters

| Adapter | Device command | Artifact | Native lifecycle |
| --- | --- | --- | --- |
| Mender | `mender-update` or legacy `mender` | `.mender` | install, commit, rollback |
| RAUC | `rauc` | `.raucb` | bundle verification, install, mark-good, mark-bad |
| OSTree | `ostree` | signed static delta | verify, apply-offline, deploy, rollback |
| SWUpdate | `swupdate` | `.swu` | dry validation and install |
| Flatpak | `flatpak` | repository ref/commit | remote validation, update, commit rollback |

The agent advertises `ota:<adapter>` only when the corresponding command is
installed. The cloud rejects a campaign if any target does not advertise the
selected capability.

Mender and RAUC still require their normal A/B partition and bootloader
integration. OSTree requires an initialized system repository. SWUpdate must be
built with the handlers, signed-image support, hardware compatibility, and
bootloader transaction behavior required by the product. When SWUpdate needs a
device-specific rollback command, provide it through a plugin rather than
assuming a particular bootloader environment.

A root-owned plugin with the same name as a built-in adapter overrides the
built-in behavior. This is the recommended way to connect SWUpdate bootloader
confirmation or a vendor-specific RAUC/OSTree layout without weakening the
cloud protocol.

## Device Configuration

Set a product identifier that exactly matches release campaigns:

```yaml
ota:
  enabled: true
  product: imx8mp-kiosk
  state_dir: /var/lib/edge-agent/ota
  staging_dir: /var/lib/edge-agent/ota/staging
  trusted_keys_dir: /etc/edge-agent/trusted-update-keys
  plugin_dir: /usr/libexec/edge-agent/ota
  max_artifact_bytes: 4294967296
  min_free_bytes: 268435456
  download_timeout: 2h
  health_timeout: 2m
  health_check_command: /usr/local/sbin/soul-room-health-check
  auto_reboot: true
```

`health_check_command` is local device configuration. It is never accepted from
the cloud. The command must return zero only when the updated device is ready to
be committed. Keep it root-owned and test it against failed services, missing
mounts, unavailable applications, and degraded hardware.

## Release Signing

Soul Room requires an outer Ed25519 signature for every OS artifact in addition
to the updater's native signature. Generate a release key outside the device:

```bash
openssl genpkey -algorithm Ed25519 -out release-private.pem
openssl pkey -in release-private.pem -pubout -out production-2026.pem
```

Install only the public key on the device:

```bash
sudo install -D -m 0644 production-2026.pem \
  /etc/edge-agent/trusted-update-keys/production-2026.pem
```

Produce the campaign values:

```bash
DIGEST="$(sha256sum release.raucb | cut -d ' ' -f1)"
SIGNATURE="$(printf %s "$DIGEST" | openssl pkeyutl -sign -rawin \
  -inkey release-private.pem | openssl base64 -A)"
```

Enter `DIGEST`, `SIGNATURE`, and `production-2026` in the campaign form. Keep the
private key in a CI signing service or HSM; never place it on a device, in the
control-plane environment, or in Git.

## Custom Adapter Plugin

Any product-specific updater can be exposed as a root-owned executable named
after the adapter:

```text
/usr/libexec/edge-agent/ota/my-updater
```

Names must match `[a-z][a-z0-9-]{1,31}`. On the next inventory report the device
advertises `ota:my-updater`, and `my-updater` can be selected in the console.

The executable receives one action argument:

```text
requires-reboot
check
install
activate
version
commit
rollback
```

For every action except `requires-reboot`, the complete update request is JSON
on standard input. `SOULROOM_OTA_ARTIFACT` contains the verified local artifact
path and `SOULROOM_OTA_PREVIOUS_VERSION` is set during rollback. Print the
installed version for `version`. Return nonzero on any failure. For
`requires-reboot`, print exactly `false` only when no reboot is required; all
other responses are treated as requiring a reboot.

The plugin is local trusted code. It is not downloaded from the cloud and no
campaign field is interpreted as a command line. It must implement an
idempotent install operation and a real rollback for production use.

## Reboot Recovery

Before requesting a reboot, the engine writes the transaction and current Linux
boot ID under `/var/lib/edge-agent/ota`. The cloud keeps the job in `rebooting`.
After systemd starts the agent on the new boot, the same job resumes at health
confirmation instead of reinstalling the artifact. Failed health checks invoke
the adapter rollback and use the same persisted reboot process.

Generic file deployment remains separate from transactional OS OTA.
