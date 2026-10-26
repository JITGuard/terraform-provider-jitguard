---
page_title: "Changelog - terraform-provider-jitguard"
subcategory: "Guides"
description: |-
  Versioned, dated release notes and the versioning/deprecation policy for the
  JIT Guard Terraform provider and the JIT Guard public API.
---

# Changelog

Every release is versioned and dated. The provider follows
[Semantic Versioning](https://semver.org/); the public API is versioned by
path. This page covers both, and links to the machine-generated GitHub release
notes for the complete change list.

## Versioning and deprecation policy

### Terraform provider

- **Patch** (`x.y.Z`) — bug fixes, documentation-visible corrections, and
  internal changes with no schema or behaviour change.
- **Minor** (`x.Y.0`) — new resources, data sources, attributes, or
  backwards-compatible behaviour.
- **Major** (`X.0.0`) — removed or renamed attributes, changed defaults, or any
  change that can break an existing configuration or state.

Deprecations are announced in this changelog and, where Terraform supports it,
in the schema itself at least one minor release before removal. The notice
names the replacement attribute or resource and the release that will remove
the old one. Upgrade notes for a breaking change are listed in that release's
entry before the provider is upgraded.

The provider is released from `main` with `semantic-release`; commit types map
to versions (`fix`/`perf` → patch, `feat` → minor, `BREAKING CHANGE`/`!` →
major). The full generated notes and signed artifacts live on the
[GitHub releases page](https://github.com/JITGuard/terraform-provider-jitguard/releases).

### JIT Guard public API

The provider targets the versioned public API at `/api/v1/customer/api/...`.
Within a major API version, JIT Guard makes additive, backwards-compatible
changes only; removing or changing the meaning of a path, field, or response
code requires a new major path such as `/v2`.

The unversioned legacy paths were deprecated on 25 October 2026 and remain
available until **Sunset: 2027-10-25**. Legacy responses carry `Deprecation`,
`Sunset`, and `Link` headers. Provider `v1.1.0` and later call `/v1`; it does
not fall back to the deprecated paths, so the API sunset does not require a
provider upgrade by itself. The public release notes are at
[jitguard.com/changelog](https://jitguard.com/changelog).

## 1.1.0 — 25 October 2026

### Changed

- Use the versioned `/api/v1/customer/api/...` public API base path instead of
  the deprecated unversioned alias. The provider `endpoint` remains the API
  origin (for example `https://app.jitguard.com`), so existing configurations
  need no change.

## 1.0.0 — 24 October 2026

### Breaking changes

- Removed `default_permission_set_arn` from
  `jitguard_organization_settings`. Configurations that set it now fail with an
  unsupported-argument error. Permission sets are chosen per promotion request
  and matched by `jitguard_approval_rule`; remove the attribute from your
  configuration before upgrading.

## 0.3.1 — 24 October 2026

### Fixed

- Default `api_key_version` when importing
  `jitguard_integration_pagerduty`, so imports no longer plan a spurious
  rotation.

## 0.3.0 — 24 October 2026

### Added

- `jitguard_organization_settings` resource for the organization name and
  approval expiry window.
- `jitguard_integration_pagerduty` resource for the PagerDuty integration.

## 0.2.0 — 24 October 2026

### Added

- Registry-ready release artifacts, including Windows builds, published under
  the `jitguard/jitguard` address.

## 0.1.0 — 21 September 2026

### Added

- Initial provider release with approval-rule management and the GitHub
  Releases distribution path.
