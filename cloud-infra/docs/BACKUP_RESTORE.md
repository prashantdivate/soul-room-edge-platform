# Backup And Restore

The local Compose maintenance profile backs up:

- the Soul Room application data volume
- the ShellHub PostgreSQL database
- ShellHub signing keys

Backups are written to <code>cloud-infra/backups</code>, which is ignored by
Git.

## Create A Backup

Run from the repository root while Docker is available:

~~~bash
docker compose -f cloud-infra/compose.yaml \
  --profile maintenance run --rm backup
~~~

The command prints a timestamp such as <code>20260916153000</code>. Keep the two
files with that timestamp together:

~~~text
cloud-infra/backups/shellhub-20260916153000.sql
cloud-infra/backups/data-20260916153000.tgz
~~~

Store customer backups outside the source checkout and protect them as
sensitive data. They contain account, device, certificate, and remote-access
state.

## Restore A Backup

Restore into a clean local installation. This removes the current Docker
volumes before loading the selected backup, so confirm that the backup files
exist and use a maintenance window:

~~~bash
docker compose -f cloud-infra/compose.yaml down -v
docker compose -f cloud-infra/compose.yaml up -d shellhub-postgres
docker compose -f cloud-infra/compose.yaml \
  --profile maintenance run --rm \
  -e BACKUP_SET=20260916153000 restore
./platform.sh up
~~~

On Windows, replace <code>./platform.sh up</code> with
<code>platform.cmd up</code>.

The <code>down -v</code> step permanently erases the current local
installation. Restore into a compatible Soul Room and ShellHub version.
Afterward, verify:

1. owner login
2. users and roles
3. enrolled devices and certificate trust
4. telemetry and job history
5. enrollment-token creation
6. ShellHub login, namespace, device acceptance, and SSH access
7. audit history

Practice restore on a separate non-production installation. A backup that has
not been restored and checked is not yet a proven recovery path.
