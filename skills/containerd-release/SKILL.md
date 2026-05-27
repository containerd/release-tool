---
name: containerd-release
description: Standardized workflow for preparing containerd patch releases, including release-tool installation, PR labeling, highlights preparation via release-note blocks, and version updates with the +unknown suffix.
---

# containerd Release Preparation

This skill provides a structured workflow for preparing patch releases on the `release/x.y` branches of containerd.

## Workflow Overview

1.  **Branch Setup**: Checkout the target `release/x.y` branch and ensure it is up to date with origin.
2.  **Audit PRs**: Review all PRs merged since the last tag.
3.  **Research and Propose**: Identify PRs that need labels or `release-note` blocks. **Present a table of proposed changes and obtain user approval before acting.**
4.  **Labeling & Highlights**: Apply approved changes to PRs.
5.  **TOML Preparation**: Create a new version TOML file in `releases/`.
6.  **Categorization & Mailmap**: Assign `area/*` or `platform/*` labels and update mailmap if needed.
7.  **Version Update**: Update `version/version.go` with the new version string, appending the `+unknown` suffix (e.g., `2.0.8+unknown`).
8.  **Verification**: Run the `release-tool` to generate and verify release notes.
9.  **Environment Check**: Run `make clean-vendor` to ensure no vendor changes are pending.
10. **Commit**: Commit the version update and TOML file using the standard commit message.

## 1. Finding and Running the release-tool

The `release-tool` is maintained as a separate repository. If it is not in your `PATH`, you can install it using `go install`:

```bash
# Install the latest version from the main branch
go install github.com/containerd/release-tool@main
```

If you have a local clone of the `release-tool` repository, you can also build it from source from its root directory:

```bash
cd /path/to/release-tool
go build -o /tmp/release-tool .
```

### Running the Tool
Run a dry run with highlights and linkified changes to verify the notes:

```bash
# Ensure GITHUB_ACTOR and GITHUB_TOKEN are set for PR data extraction
export GITHUB_ACTOR=$(gh api user -q .login)
export GITHUB_TOKEN=$(gh auth token)

# Run release-tool (Note: the -t flag for the tag should NOT include +unknown)
# The -n flag is required to see output in the console.
# Save the output to a notes file in the root directory for review.
release-tool -r -n -g -l -t v2.2.3 ./releases/v2.2.3.toml | tee v2.2.3-notes.md
```

- `-r`: Refresh cache (critical if you recently updated PR labels or bodies).
- `-n`: Dry run (output to stdout).
- `-g`: Include highlights extracted from PRs.
- `-l`: Linkify changelog commits/PRs.

## 2. Proposing Changes (Mandatory)

Before modifying any pull request, you **must** present a table of proposed changes to the user for approval.

### Required Table Format:
| PR # | Current Title | Labels to Add | Proposed `release-note` block | Rationale |
| :--- | :--- | :--- | :--- | :--- |
| #12345 | `[release/2.0] fix: bug` | `impact/changelog`, `area/cri` | `Fix bug in CRI plugin` | Highlight. Adds label and categorization. |

**Stop and ask for confirmation after presenting this table.** Do not use `gh pr edit` until the user provides an explicit "Yes" or "Proceed."

## 3. Highlights and Labels

Highlights are extracted automatically by the `release-tool` based on PR data.

### Requirements for a Highlight:
1.  **Label**: The PR must have the `impact/changelog` label.
2.  **Category**: The PR should have an `area/*` label (e.g., `area/cri`, `area/runtime`, `area/snapshotters`) or `platform/*` label (e.g., `platform/windows`) for categorization.
3.  **Note**: The tool extracts notes from a `release-note` markdown block in the PR description.

### Tracing Context for Cherry-picks:
When auditing PRs on a release branch, they are often cherry-picks. **Always trace the cherry-pick back to the original PR** (e.g., on `main`) to understand its full context.

#### Example:
1. **Identify the original PR**: View the cherry-pick PR's description.
   ```bash
   gh pr view 13198 --json body
   # Output: {"body":"This is an automated cherry-pick of #13186\n\n/assign fuweid"}
   ```
2. **Inspect the original PR**: Read the original PR's description and comments to understand the "why" and generate a better release note.
   ```bash
   gh pr view 13186 --json title,body,labels
   # This reveals that #13198 fixes "failed to mount ... internal mount option X-containerd.mkfs.fs=ext4 was not consumed"
   # which is much more descriptive than the cherry-pick's title.
   ```

- Read the original PR's description and comments to generate a better, more descriptive release note.
- **Sibling Cherry-Pick Audit:** Check if the same change has been cherry-picked to other active release branches (e.g., `release/2.x`). Sibling PRs on other branches might have more polished `release-note` blocks or valuable review comments. **Always ensure consistency in the release note wording across all active release branches for the same change.**
  > [!IMPORTANT]
  > **Do not rely solely on the sibling release preparation PR's description** (which may be truncated in logs or incomplete). **Query the individual sibling PRs directly** (e.g., using `gh pr view <sibling-number>`) to inspect their exact `release-note` blocks and ensure you use the most polished version.
- Check if it was associated with a security advisory or if it was determined **not** to be a vulnerability (e.g., "security hardening" vs. "vulnerability fix").
- **Crucial Distinction:** We only "fix" vulnerabilities that exist directly within containerd or our dependencies. We do **not** "fix" vulnerabilities in external components like the Linux kernel. If a PR adds mitigation for an external vulnerability (like a kernel CVE), it must be described as "hardening" the policy/component against the issue, not fixing it.

### What NOT to highlight:
Do **not** add the `impact/changelog` label to the following types of PRs:
- **Toolchain Updates**: Go version bumps (unless specifically requested), CI workflow changes, or linter updates. Go toolchain updates are generally not highlighted because not all builds of containerd use identical toolchains (e.g., distro packaging), making these updates non-universal.
- **Standard Dependency Bumps**: Standard library or external dependency updates (unless they contain a relevant security fix or significant feature).
- **Internal Maintenance**: Changes to `CODEOWNERS`, `README.md`, or repository metadata.
- **Minor Hardening**: Security hardening that does not fix an active vulnerability or significant user-facing issue.

### Setting the release-note block:
Use `gh pr edit` to append a markdown block to the end of the PR body. Do not change the PR title.

**Format:**
```markdown
```release-note
<Description starting with present-tense verb>
```
```

### Writing Style:
- Use short, descriptive phrases starting with a present-tense verb.
- **Avoid Redundant Subject Prefixes:** Do not include redundant package or area prefixes (like `runtime:`, `cri:`, `seccomp:`, `apparmor:`) in the release-note block. Since the `release-tool` automatically categorizes highlights under headers like `#### Runtime` or `#### Container Runtime Interface (CRI)`, these prefixes are redundant. Start directly with the verb or integrate the component name naturally into the description.
  * *Incorrect:* `runtime: Support both "volatile" and "fsync=volatile" mount options...`
  * *Correct:* `Support both "volatile" and "fsync=volatile" mount options...`
  * *Incorrect:* `apparmor: Set abi conditionally to support AppArmor versions < 3.0`
  * *Correct:* `Set AppArmor abi conditionally to support versions < 3.0`
- For security-hardening fixes, explicitly use "Apply hardening to..." (e.g., `Apply hardening to block AF_ALG in default socket policy`).
- For compatibility fixes, frame the description around "support" and compatibility rather than "breaking" to avoid causing unnecessary alarm to users (e.g., `Set AppArmor abi conditionally to support versions < 3.0` instead of `avoid breaking AppArmor versions < 3.0`).
- **Example**: `Enable mount manager in diff walking to fix layer extraction errors with some snapshotters (e.g., EROFS)` (Correct)
- **Focus on Impact (What, not How):** Describe *what* the bug was and its impact on the user/system, rather than *how* the code was changed. Explain the problem that was solved.
  * *Incorrect:* `Forward netns_path, rootfs, and annotations in sandbox Create`
  * *Correct:* `Fix sandbox creation via gRPC proxy dropping configuration fields`

## 4. TOML Preparation and Security Updates

Create a new file `releases/v<VERSION>.toml`.

### Correct TOML Formatting:
Ensure the TOML file follows the standard project format, including the correct `ignore_deps` and `postface` sections.
- **Preface Consistency:** Keep the preface wording consistent with past releases (e.g., "contains various fixes and improvements"). Do **not** call out Go toolchain updates or minor changes in the preface; reserve the preface for notable changes or actual containerd/dependency security updates.

```toml
# commit to be tagged for new release
commit = "HEAD"

project_name = "containerd"
github_repo = "containerd/containerd"
match_deps = "^github.com/(containerd/[a-zA-Z0-9-]+)$"
ignore_deps = [ "github.com/containerd/containerd" ]

# previous release
previous = "v2.2.2"

pre_release = false

preface = """\
The third patch release for containerd 2.2 contains various fixes
and updates including a security patch.

### Security Updates

* **spdystream**
  * [**CVE-2026-35469**](https://github.com/moby/spdystream/security/advisories/GHSA-pc3f-x583-g7j2)
"""

postface = """\
### Which file should I download?
* `containerd-<VERSION>-<OS>-<ARCH>.tar.gz`:         ✅Recommended. Dynamically linked with glibc 2.35 (Ubuntu 22.04).
* `containerd-static-<VERSION>-<OS>-<ARCH>.tar.gz`:  Statically linked. Expected to be used on Linux distributions that do not use glibc >= 2.35. Not position-independent.

In addition to containerd, typically you will have to install [runc](https://github.com/opencontainers/runc/releases)
and [CNI plugins](https://github.com/containernetworking/plugins/releases) from their official sites too.

See also the [Getting Started](https://github.com/containerd/containerd/blob/main/docs/getting-started.md) documentation.
"""
```

### Security Updates
If the release contains security fixes (either for containerd itself or for dependencies), add them manually to the `preface` section:
- For dependency updates, list the dependency name as the bullet (e.g., `spdystream`).
- For containerd's own vulnerabilities, list `containerd` as the bullet.
- Use the CVE ID as the link text pointing to the advisory.
- **Pre-publication Links:** Include the link to the GitHub Security Advisory (GHSA) even if the advisory is currently private/draft. It will become public simultaneously with the release publication, ensuring the links work for users immediately upon release.

**Example:**
```toml
preface = """\
The first patch release for containerd 2.3 contains various fixes
and updates including a security patch.

### Security Updates

* **containerd**
  * [**CVE-2026-46680**](https://github.com/containerd/containerd/security/advisories/GHSA-fqw6-gf59-qr4w)

* **spdystream**
  * [**CVE-2026-35469**](https://github.com/moby/spdystream/security/advisories/GHSA-pc3f-x583-g7j2)
"""
```

## 5. Commit Messages

Always check the historical commit messages for the current release branch. Always use `git commit -s` to commit your changes. This automatically appends the correct `Signed-off-by` line based on your git configuration, which is required for all containerd contributions. Do not manually construct the `Signed-off-by` line.

For patch releases, the standard message is:

```text
Prepare release notes for vX.Y.Z

Signed-off-by: Your Name <your@email.com>
```

## 6. PR Categorization (Areas)

Assign appropriate labels using `gh pr edit --add-label`. Verify label names with `gh label list`. See [references/categorization.md](references/categorization.md) for comprehensive guidelines.
- `area/cri`
- `area/runtime`
- `area/snapshotters`
- `platform/windows` (Note: use `platform/` prefix for Windows)

- **Avoid Redundant Area Labels:** Ensure a PR has only one primary `area/*` or `platform/*` label unless it genuinely spans multiple distinct areas. Redundant area labels (e.g., having both `area/runtime` and `area/storage` for a storage-specific fix) will cause the PR to be listed multiple times in different sections of the generated release notes. Check the generated notes during verification to ensure no highlights are duplicated.

### Dependency Repositories
When auditing a dependency repository (e.g., `containerd/nri`, `containerd/ttrpc`), you must also apply `impact/changelog`, `area/*`, and `release-note` blocks to its PRs so they are included in the main containerd release notes when the dependency is updated.
- Use the `--repo <org>/<repo>` flag with `gh` commands if you are not in a local clone of that dependency.
- Use an area label corresponding to the dependency (e.g., `area/nri` for the `containerd/nri` repository).

## 7. Mailmap and Contributors

See [references/mailmap.md](references/mailmap.md).

When adding new contributors to the `.mailmap` file, ensure that they are manually inserted into their correct alphabetical position. Do not use tools like `sort` on the whole file, as the existing file might not perfectly match simple sorting algorithms and could result in unnecessary diffs.

## 8. Verification

Before finalizing, always check:
1.  `version/version.go` contains the exact version string **with the `+unknown` suffix** (e.g., `2.2.3+unknown`).
2.  `make clean-vendor` results in a clean `git status vendor/`.
3.  The `release-tool` output matches expectations and contains all intended highlights with correct wording.
4.  The generated notes file (e.g., `vX.Y.Z-notes.md` in the root directory) has been created and reviewed, but is **NOT** staged or committed.
