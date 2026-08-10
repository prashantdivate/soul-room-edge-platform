# Security Limitations

This MVP does not include a concrete TPM, secure-element, SQLite, generated
protobuf, production OTA, or remote-access implementation.

The local development simulator accepts development enrollment tokens and is
not a production control plane.

Do not expose Docker policy exceptions or downstream write commands without a
separate threat review and customer-specific safety analysis.
