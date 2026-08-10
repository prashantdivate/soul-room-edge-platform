# Shared Agent Protocol

The protobuf contract is defined in `api/proto/fleet/v1/fleet.proto`.

Every device-facing message uses an `Envelope` with:

* protocol version
* message ID
* correlation ID
* tenant ID
* device ID
* gateway ID when applicable
* sequence number
* timestamp
* payload schema version
* payload digest

Runtime MVP endpoints accept JSON representations of these messages over HTTPS.
The gateway validates payload size, timestamp skew, digest, tenant/device
identity, and duplicate message IDs.
