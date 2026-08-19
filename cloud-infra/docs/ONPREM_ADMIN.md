# On-Premises Administration

## Layered configuration

Soul Room separates configuration by risk and lifecycle:

1. Built-in defaults make a clean Compose installation usable without an
   `.env` file.
2. Optional `SOULROOM_*` environment values provide deployment defaults.
3. Organization owners can override safe operational defaults from **Platform >
   On-prem admin > Platform settings**. These values are tenant-scoped,
   persistent, effective immediately, and audit recorded.

The UI controls the organization display name and domain, device offline
window, initial telemetry range, OTA pilot percentage, enrollment-token
lifetime, and queued-job expiry. **Restore defaults** deletes the tenant's UI
override and returns to the environment or built-in values.

Secrets and infrastructure settings are deliberately not editable in the web
console. Database and object-store credentials, CA/private keys, bootstrap admin
credentials, bind addresses, public gateway hosts, ShellHub endpoints, payload
limits, and session lifetime stay environment-managed. The UI shows a safe,
read-only summary without returning secret values.

## Operational runbooks

Administrators should manage certificate replacement, password recovery,
migration rollback, backups, restore drills, local registry setup, DNS, NTP, and
offline package/image bundle processes through documented runbooks.
