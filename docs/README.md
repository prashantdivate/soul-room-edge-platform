# Soul Room Documentation

This page is the shortest route from a question to the guide that answers it.
Start with the repository [README](../README.md) for the first platform launch
and first device enrollment.

## Platform Operations

| I want to... | Read |
| --- | --- |
| Start Soul Room on Windows, Linux, macOS, or WSL | [Run the platform locally](../cloud-infra/docs/RUNNING_LOCALLY.md) |
| Configure organization defaults and deployment settings | [On-premises administration](../cloud-infra/docs/ONPREM_ADMIN.md) |
| Configure bundled remote access | [ShellHub integration](../cloud-infra/docs/SHELLHUB_INTEGRATION.md) |
| Back up or restore a local installation | [Backup and restore](../cloud-infra/docs/BACKUP_RESTORE.md) |
| Understand the services and trust boundaries | [System architecture](../cloud-infra/docs/SYSTEM_ARCHITECTURE.md) |
| Understand stored records | [Data model](../cloud-infra/docs/DATA_MODEL.md) |
| Review API and agent messages | [Shared protocol](../cloud-infra/docs/PROTOCOL.md) |
| Review tenant and platform security assumptions | [Cloud threat model](../cloud-infra/docs/THREAT_MODEL.md) |
| Check what is implemented or deferred | [Cloud implementation status](../cloud-infra/docs/IMPLEMENTATION_STATUS.md) |

## Device Agent

| I want to... | Read |
| --- | --- |
| Build, deploy, configure, and verify an agent | [Agent guide](../agent/README.md) |
| Review detailed Linux installation | [Embedded Linux installation](../agent/docs/EMBEDDED_LINUX_INSTALLATION.md) |
| Install on Ubuntu or Debian | [Debian installation](../agent/docs/DEBIAN_INSTALLATION.md) |
| Add the agent to a Yocto image | [Yocto integration](../agent/docs/YOCTO_INTEGRATION.md) |
| Run it on an i.MX8MP device | [Yocto i.MX8MP example](../agent/docs/RUNNING_ON_YOCTO_IMX8MP.md) |
| Configure Mender, RAUC, OSTree, SWUpdate, Flatpak, or a custom updater | [OTA adapters](../agent/docs/OTA_ADAPTERS.md) |
| Diagnose enrollment or connectivity | [Agent troubleshooting](../agent/docs/TROUBLESHOOTING.md) |
| Understand offline buffering | [Offline queue policy](../agent/docs/QUEUE_POLICY.md) |
| Add a downstream-device connector | [Connector SDK](../agent/docs/CONNECTOR_SDK.md) |
| Review device-side security assumptions | [Agent threat model](../agent/docs/THREAT_MODEL.md) and [security limitations](../agent/docs/SECURITY_LIMITATIONS.md) |
| Check what is implemented or deferred | [Agent implementation status](../agent/docs/IMPLEMENTATION_STATUS.md) |

## Design References

- [Agent architecture](../agent/docs/ARCHITECTURE.md)
- [Agent architecture decisions](../agent/docs/ARCHITECTURE_DECISIONS.md)
- [Job state machine](../agent/docs/JOB_STATE_MACHINE.md)
- [Upgrade compatibility policy](../agent/docs/UPGRADE_POLICY.md)
- [Cloud architecture decisions](../cloud-infra/docs/ARCHITECTURE_DECISIONS.md)
- [Quotas and abuse prevention](../cloud-infra/docs/QUOTAS.md)
- [Remote-access security boundary](../cloud-infra/docs/REMOTE_ACCESS.md)

Documentation should describe behavior that exists in the repository. Planned
work belongs in the implementation plans and status matrices, not in setup
instructions.
