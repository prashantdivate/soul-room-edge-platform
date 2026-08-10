# Backup and Restore

Backups must include:

* PostgreSQL logical or physical backup
* object-storage bucket backup
* service configuration
* CA and secret backup guidance

Run `scripts/backup-compose.sh` for the Compose profile. Run
`scripts/restore-compose.sh` in a clean environment and validate login,
enrollment-token creation, telemetry query, and audit history before treating a
backup as complete.
