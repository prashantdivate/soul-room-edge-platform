# Protocol

The versioned protocol contract is defined in `api/proto/edge/v1/edge.proto`.
The MVP runtime serializes equivalent JSON envelopes over HTTPS while protobuf
binding generation is deferred.

Every envelope includes protocol version, message ID, tenant ID, device ID,
sequence, creation timestamp, payload schema version, payload digest, and
payload bytes. Receivers validate timestamp skew and digest before accepting
messages.

Message families include enrollment, hello, heartbeat, presence, inventory,
telemetry batch and acknowledgement, job delivery and result, file metadata,
application state, container state, OTA state, downstream registration,
downstream telemetry, connector health, credential rotation, agent configuration
updates, and capability negotiation.
