# On-Premises Deployment

## Small Installation

Use the root `compose.yaml` for API, device gateway, worker, web UI,
PostgreSQL, MinIO, and mock SMTP. Recommended minimum: 4 vCPU, 8 GiB RAM, 100 GiB
SSD, reliable NTP, local DNS, and regular backups.

## Scalable Installation

Use `deploy/helm` with external PostgreSQL and S3-compatible storage. Configure
ingress TLS, secrets, resource requests/limits, persistent volumes, network
policies, readiness/liveness probes, and pod disruption budgets.

## Offline/Air-Gapped Limits

Air-gapped installation is not claimed complete until every image, dependency,
chart, migration, and restore test is packaged and validated.
