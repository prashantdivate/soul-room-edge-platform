# Container Security Policy

Managed container operations are restricted to platform-managed services.

Rejected by default:

* privileged mode
* host networking
* arbitrary host path mounts
* Docker socket mounts
* missing image references
* unmanaged services

Policy exceptions must be explicit administrator configuration and should be
audited. Registry passwords must never be logged.
