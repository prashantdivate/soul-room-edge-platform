# Threat Model

## Assets

* Device private keys and certificates.
* Enrollment tokens.
* Job authorization and execution state.
* Telemetry, inventory, and downstream device metadata.
* Container registry credentials.
* File deployment artifacts and signatures.

## Primary Threats

* Fleet-wide compromise through shared credentials.
* Man-in-the-middle attacks against the control-plane connection.
* Replay of old commands or telemetry acknowledgements.
* Arbitrary shell or file-write command injection.
* Unsafe downstream controller writes.
* Container privilege escalation through policy exceptions.
* OTA downgrade or artifact tampering.

## Controls

* Per-device locally generated keys.
* TLS validation and private CA support.
* Message IDs, timestamps, nonces, sequence counters, and job expiry.
* Typed allowlisted jobs only.
* File deployment allowlists, checksum and signature hooks, atomic staging.
* Read-only connector defaults.
* Container policy rejects privileged mode, host networking, arbitrary mounts,
  and Docker socket mounts by default.
* OTA adapter requires compatibility, digest, signature, health confirmation,
  and explicit rollback.

## Limitations

* TPM and secure-element implementations are interfaces in this MVP.
* SQLite and protobuf generated bindings are deferred.
* Remote access tunnel is not implemented.
