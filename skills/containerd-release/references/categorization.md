# containerd PR Categorization Guide

This guide provides examples for categorizing pull requests and determining if they should be highlights.

## Area Mapping

### `area/cri`
**Scope**: Kubernetes Container Runtime Interface (CRI) implementation.
- **Examples**:
  - Fixes to pod sandbox creation or deletion.
  - Image pull behavior within the CRI plugin.
  - Changes to CRI-specific metrics.

### `area/runtime`
**Scope**: Core container runtime behavior, task management, and shims.
- **Examples**:
  - Fixes to `containerd-shim` (v2).
  - SELinux policy updates.
  - AppArmor or Seccomp profile changes.

### `area/snapshotters`
**Scope**: Snapshotters and their drivers (e.g., `overlayfs`, `devmapper`, `btrfs`).
- **Examples**:
  - Fixes to snapshotter mount options or extraction logic.
  - Remote snapshotter integrations.

### `area/storage`
**Scope**: Content store, metadata database, garbage collection, and image storage.
- **Examples**:
  - Content store garbage collection.
  - Metadata database fixes.

### `area/toolchain`
**Scope**: Build system, CI/CD, and development environment.
- **Note**: These are **rarely** highlights.

### `area/ctr`
**Scope**: The `ctr` command-line development tool.

## Security-related Changes

Distinguishing between vulnerability fixes and security hardening is critical for accurate highlights.

| Type | Description | Highlight? |
| :--- | :--- | :--- |
| **Vulnerability Fix** | Fixes a bug that has a CVE or GHSA (e.g., GHSA-pwhc-rpq9-4c8w). | **Yes**. Usually categorized in a "Security Updates" section if available, otherwise a highlight. |
| **Security Hardening** | Proactive improvement to security posture, but not a fix for an exploitable vulnerability. | **Maybe**. Only if it significantly changes behavior or fixes a visible user issue. If determined not to be a vuln (check original PR discussion), it is often excluded from highlights. |

### How to Verify:
1.  **Follow the Chain**: Check the current PR's description for "Backport of #XXXXX" or "Cherry-pick of #XXXXX".
2.  **Read the Source**: Open the original PR (#XXXXX) and read the description and all maintainer comments.
3.  **Check for CVE/GHSA**: If there is no CVE or GHSA mentioned in the original PR or its related issues, it is likely hardening.

## Best Practices
1.  **Prefer Specificity**: If a change affects both core runtime and CRI, but is primarily motivated by a CRI bug, use `area/cri`.
2.  **Highlight Selection**: Only use `impact/changelog` for user-facing bug fixes or significant feature improvements. **Avoid highlighting dependency updates or minor hardening.**
3.  **Check Previous Releases**: If unsure, look at how similar PRs were categorized in previous patch releases.
