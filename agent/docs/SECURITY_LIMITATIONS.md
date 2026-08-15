# Security Limitations

This MVP does not include a concrete TPM, secure-element, SQLite, or generated
protobuf implementation. The OTA transaction engine is implemented, but each
device image still requires qualified native updater, partition, bootloader,
power-loss, and rollback integration before commercial deployment. Interactive remote access is not
implemented in `edge-agent`; it is deliberately delegated to the separately
installed ShellHub agent and server stack.

The local development simulator accepts development enrollment tokens and is
not a production control plane.

Do not expose Docker policy exceptions or downstream write commands without a
separate threat review and customer-specific safety analysis.

OTA adapters execute with system privileges because operating-system updaters
must write protected storage and boot state. Keep the agent configuration,
trusted release keys, health command, and plugin directory root-owned. Never
install an adapter received through a fleet job.
