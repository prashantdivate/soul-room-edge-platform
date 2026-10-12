# Threat Model

| Threat | Mitigations | Residual Risk |
| --- | --- | --- |
| Compromised device | Per-device identity, certificate revocation, scoped jobs, telemetry quotas | Device-originated false telemetry until detected |
| Stolen enrollment token | Random one-time tokens, expiry, tenant/profile scope, audit, revocation | Token can enroll once before expiry if stolen |
| Cloned device identity | Unique certs, duplicate fingerprint detection, revocation, audit | Offline clone may appear until reconnect analysis |
| Malicious tenant user | RBAC, audit, quotas, server-side tenant checks | Authorized destructive action can still harm own tenant |
| Compromised org admin | Approval workflows for risky actions, immutable audit, least privilege | Org admin can affect own fleet |
| Cross-tenant API access | Tenant-scoped repositories, RBAC tests, optional PostgreSQL RLS | Application bugs remain possible |
| Replayed telemetry | Message IDs, timestamps, sequence checks, duplicate cache | Large replay floods still consume rate-limit budget |
| Replayed jobs | Job ID idempotency, expiry, attempt tracking, device result correlation | Clock skew can require operational handling |
| Stale offline jobs | Expiry enforcement at API and device gateway | Long offline periods can delay safe operations |
| Malicious artifact | Digests, signatures, compatibility metadata, scanning hook, no public URLs | Scanner coverage depends on deployment |
| Malicious Compose manifest | Policy rejects privileged mode, host namespaces, host network, Docker socket | Approved exceptions carry tenant-specific risk |
| Object-storage exposure | Short-lived signed URLs, tenant metadata, private buckets | Misconfigured external S3 can leak data |
| Database compromise | Least privilege, backups, audit, secret separation, optional encryption | Full DB compromise exposes metadata |
| CA compromise | CA key storage guidance, rotation, revocation, external PKI option | Active attacker can issue device certs until rotation |
| Reverse-proxy compromise | TLS to services where practical, secure headers, audit | Traffic metadata visible at proxy |
| Insider access | Audit logs, least privilege, tenant scope, break-glass docs | Platform admin remains high-trust role |
| Credential leakage | No token logs, secure cookies, revocation, secret scanning | Client endpoint compromise can steal active sessions |
| Telemetry DoS | Payload limits, quotas, rate limits, backpressure | Distributed device flood requires infrastructure controls |
| Reconnect storm | Jitter guidance, rate limits, stateless gateway, durable DB state | Cloud load balancer capacity may be exhausted |
| Gateway impersonation | mTLS, parent-child authorization, enrollment policy | Stolen gateway cert can spoof children until revoked |
| Downstream command abuse | Connector command allowlists, audit, approval hooks | Safe command definitions are connector-specific |
| Prompt injection through fleet data | Fleet values are marked untrusted, compact context excludes raw logs, server validates every proposed action | A model can still produce misleading analysis that an operator must review |
| AI action overreach | No tool loop, narrow typed-job allowlist, current RBAC recheck, ten-minute actor/tenant-bound approval, idempotency, audit | An authorized operator can approve a harmful reboot or collection at the wrong time |
| AI provider credential theft | AES-256-GCM at rest, tenant-bound authenticated data, redacted API responses, restrictive key-file permissions | Host or volume compromise can expose both ciphertext and its encryption key |
| AI provider data retention | Minimized tenant snapshot and explicit exclusions | The external provider controls retention and jurisdiction under its own terms |
| Custom model endpoint abuse | Owner-only configuration, URL validation, metadata/link-local rejection, bounded response and timeout | A trusted tenant owner can still target reachable private services; production deployments should enforce egress policy |
| Update rollback failure | OTA state machine, explicit rollback state, health confirmation | Adapter bugs can leave device needing manual recovery |
| Supply-chain compromise | Dependency scanning, image scanning, signed artifacts, SBOM guidance | Build system compromise remains severe |
