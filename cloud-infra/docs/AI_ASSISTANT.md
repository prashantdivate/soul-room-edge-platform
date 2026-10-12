# AI Assistant

Soul Room includes an optional, tenant-aware operations assistant. It explains
fleet state, identifies useful next checks, links operators to the relevant
workspace, and can propose a small set of typed device jobs. It is inactive
until an organization owner connects a model provider.

## Supported providers

Open **AI Assistant > Operations copilot**, then select one of:

| Provider | Connection |
| --- | --- |
| OpenAI | `POST https://api.openai.com/v1/responses` |
| Anthropic | `POST https://api.anthropic.com/v1/messages` with API version `2023-06-01` |
| Local / compatible | `POST {base_url}/chat/completions` for Ollama, LocalAI, llama.cpp gateways, or a private OpenAI-compatible gateway |

Provider model names remain editable because availability differs by account
and changes over time. API keys are entered in the console, encrypted before
being written to the platform data volume, and never returned to the browser.
Leaving the key field blank preserves the saved key.

For an Ollama-compatible service running on the same computer as Compose, use:

```text
http://host.docker.internal:11434/v1
```

The Compose topology maps `host.docker.internal` on Docker Desktop and Linux
Docker Engine. For another container in the same Compose network, use that
service's DNS name. A compatible endpoint may omit the API key; hosted OpenAI
and Anthropic connections require one.

OpenAI and Anthropic base URLs are fixed by the control plane and cannot be
replaced from the browser. Compatible URLs are administrator-controlled and
validated before use: credentials, query strings, fragments, unspecified,
multicast, and link-local addresses are rejected, redirects are disabled, and
DNS is checked again when the server connects. Private and loopback addresses
remain available so an owner can intentionally connect a self-hosted model.

## Fast, bounded requests

The assistant adds no work to the device agent and performs no background
model polling. A provider request occurs only when a user sends a message or an
owner tests the connection.

Each request contains a compact snapshot rather than raw fleet records:

* fleet totals and update/artifact counts;
* at most 20 relevant, offline, or unhealthy devices;
* only the latest CPU, memory, storage, and temperature value per device;
* at most eight recent jobs and six update campaigns;
* aggregate package counts rather than the full package inventory;
* at most six short conversation turns.

Responses are capped at 900 output tokens. The default guard allows 30 requests
per user per minute, which leaves ample interactive headroom while stopping a
runaway client. Change `SOULROOM_AI_REQUESTS_PER_MINUTE` only when the provider
and deployment capacity justify it.

Conversation history stays in the current browser session. Clearing the chat,
signing out, or closing the browser session removes it from the console. Soul
Room does not store prompt or answer text in its control-plane data or audit
log.

## Data boundary

The model receives tenant-scoped operational summaries. Soul Room deliberately
excludes:

* provider credentials and user/account records;
* device certificates and certificate subjects;
* serial numbers and remote-access identifiers;
* IP addresses and precise device locations;
* raw logs, job output, and full software-package lists.

Device names and other fleet values are treated as untrusted data in the model
instructions. Operators should still review model conclusions against the
source page before making a production decision. The configured provider's
privacy, retention, regional, and billing terms apply to every request.

## Action safety

The model cannot call a device, write the database, launch a shell, deploy an
artifact, or start an OTA campaign. It can propose only:

* navigation to an allowed Soul Room page;
* `collect_diagnostics`;
* `collect_logs`;
* `collect_inventory`;
* `scan_vulnerabilities`;
* `reboot`.

The assistant sees an `available_actions` list reported for each device and is
instructed to propose only those actions. The server enforces that list again
when it creates a proposal and when the operator approves it. Collection jobs
require the agent's `jobs` capability, advisory scans require Trivy, and reboot
requires the explicit `job:reboot` capability from a current agent.

An approved reboot is scheduled instead of being executed inline, leaving time
for the agent to report the successful job result. Systemd devices use a
transient unit with an approximately ten-second delay; other Linux systems can
fall back to `shutdown -r +1`. The agent service must have permission to
schedule a system reboot. Upgrade older installed agents before expecting the
reboot proposal to appear.

Device actions appear as a review step. The operator must open the proposal and
select **Approve and queue**. Soul Room then validates the signed, ten-minute
approval against the current tenant, user, device, permission, and job-type
allowlist. Replaying the same approval returns the same job rather than
creating a duplicate. The approval and resulting typed job are audit recorded.

## Credential storage and backup

The platform creates `/data/ai-secrets.key` with restrictive permissions and
uses AES-256-GCM with tenant-specific authenticated data to encrypt provider
keys. Both `control-plane.json` and `ai-secrets.key` are required to restore
working AI connections. They live in the same `app-data` volume used by the
Compose platform.

Do not copy `control-plane.json` without its corresponding encryption key. If
the key is lost, the ciphertext cannot be recovered; remove and reconnect the
provider from the Assistant page. Never commit the platform data volume, local
`.env`, or exported backups to Git.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Connection test returns 401/403 | Verify the provider key and that the selected model is enabled for the account |
| Connection test returns 404 | Confirm the model name and compatible endpoint includes its `/v1` base path when required |
| Local model cannot be reached | Bind the model server to an interface reachable from Docker and use `host.docker.internal`, not `localhost` |
| Assistant says configuration is unavailable | Confirm the `app-data` volume is writable and `ai-secrets.key` exists beside platform state |
| An action is not offered | Check the user's RBAC permission and the device capabilities. Upgrade the device agent; install Trivy for advisory scans |

The assistant is decision support, not an autonomous control loop. Production
change policy, approval ownership, provider governance, and incident response
remain the operator's responsibility.
