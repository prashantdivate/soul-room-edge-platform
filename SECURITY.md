# Security Policy

## Supported Version

Soul Room is currently a public beta. Security fixes are applied to the latest
code on `master` and included in the next published image and tagged release.

## Report A Vulnerability

Do not disclose suspected vulnerabilities, credentials, certificates, device
identities, or customer data in a public issue.

Use **Security > Advisories > Report a vulnerability** in this GitHub
repository. Include the affected component and version, reproduction steps,
impact, and any suggested mitigation. Reports will be acknowledged and
triaged before coordinated disclosure.

For ordinary bugs that do not expose security-sensitive information, use
GitHub Issues.

## Automated Checks

GitHub Actions runs CodeQL, `govulncheck`, npm audit, and Trivy scans for
dependencies, secrets, configuration, and final container images. High or
critical findings block image publication. These checks reduce
risk but do not replace deployment-specific threat modeling, penetration
testing, key management, TLS configuration, or OTA hardware qualification.
