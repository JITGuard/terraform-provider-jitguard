# Security Policy

## Supported versions

Only the latest published provider version receives security fixes.
Upgrade with `terraform init -upgrade` and a relaxed version constraint
(`~>` the current minor).

## Reporting a vulnerability

**Do not open a public issue.** Use GitHub's private vulnerability reporting:
go to the repository's **Security** tab and choose **Report a vulnerability**.
If you cannot use that form, contact the JIT Guard maintainers through the
support channel listed on [jitguard.com](https://jitguard.com) and ask for the
security contact.

Please include:

- the provider version and Terraform version,
- a minimal configuration or reproduction,
- the impact you believe is possible.

Never include a live `bg_sk_...` API key, AWS credentials, or customer data in
a report. Revoke any key you believe may have been exposed.

## Scope

This policy covers the provider binary and the code in this repository. The
JIT Guard service, API Gateway, and customer AWS accounts are covered by the
JIT Guard platform security programme.
