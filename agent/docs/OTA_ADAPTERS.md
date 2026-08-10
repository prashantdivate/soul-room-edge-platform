# OTA Adapter Guide

The OTA package defines adapter methods for compatibility checks, staging,
verification, installation, activation, reboot request, health confirmation,
rollback, current version, and status.

Implemented:

* Simulator adapter.

Planned production adapters:

* OSTree
* Mender
* RAUC
* SWUpdate
* Custom MCU bootloaders

Generic file deployment is intentionally separate from transactional OS OTA.
