# Minor (`.0`) Releases

Read this only when preparing a minor release (`vX.Y.0`). Patch releases do not need it.

Everything in `SKILL.md` still applies. This document covers only what differs.

## Summary of differences

| Aspect | Patch release | Minor (`.0`) release |
| :--- | :--- | :--- |
| Branch | `release/x.y`, already exists | Usually cut from `main`; `release/x.y` may not exist yet |
| TOML `previous` | Previous patch tag (e.g. `v2.3.4`) | Previous minor tag (e.g. `v2.3.0`) |
| `### Security Updates` preface | Include when applicable | **Omit entirely** |
| Bug-fix highlights | Nearly all fixes are highlighted | Only individually notable fixes |
| What dominates the notes | Fixes | Features and deprecations |

## Branch and TOML

The `release/x.y` branch typically does not exist yet when a `.0` is being prepared; work from `main` with `commit = "HEAD"`.

Set `previous` to the previous *minor* tag (e.g. `previous = "v2.3.0"`), not the newest patch on that line. This is what makes the diff range large and is the direct cause of the highlight-volume problem described below.

## No `### Security Updates` section

Do **not** add a `### Security Updates` preface to a minor release, even when `Merge commit from fork` commits appear in the range.

The section exists to tell a user what upgrading from the previous release fixes. For a patch release that is meaningful, because the previous patch was vulnerable. For a `.0` it is not: the fixes landed on `main` before the minor line ever shipped, so no released `X.Y.z` version was ever affected. Those advisories are already public, and were announced with the patch releases on the older branches that carried the backports.

Fork commits still appear in the commit log under `<details><summary>... commits</summary>`. That is the correct and sufficient disclosure for a `.0`.

## The bar for bug-fix highlights is much higher

A patch release *is* its bug fixes, so listing them all is correct. A `.0` is about the new minor line, and most of its fixes are not news to anyone.

Two structural reasons, not merely taste:

1.  **Much of the fix traffic has already been announced.** Because `previous` is the previous minor tag, every fix cherry-picked to the previous release branch after that tag resurfaces here. Check for `cherry-pick/x.y` and `cherry-picked/x.y` labels: those changes already shipped, and were already highlighted, in a patch release. They are still legitimately new for someone upgrading directly from `X.Y-1.0`, but they are a rerun for anyone tracking patches.
2.  **Fixes compete with the feature narrative.** Highlighting routine fixes buries the features and deprecations that define the release.

Historical precedent for what actually clears the bar:

| Release | Bug-fix highlights | Total highlights |
| :--- | ---: | ---: |
| 2.2.0 | ~1 | ~20 |
| 2.3.0 | ~3 | 37 |
| 2.4.0 | 2 (plus 3 hardening) | 27 |

> [!WARNING]
> These ratios are an **outcome of a selective bar, not a target**. Prior `.0` releases absorbed just as much fix traffic; they simply highlighted very little of it. Never reason that a release "has more fixes, so it should highlight more." Evaluate each fix on its own reach and severity.

When a fix is borderline, ask who actually hits it. A fix reachable only in a narrow configuration - one snapshotter, one filesystem, one isolation mode, one image layout - generally does not clear the bar for a `.0`, even when the underlying bug is severe for those who hit it.

Security *hardening* is judged separately from bug fixes: a change to default behavior that users can observe (masked paths, scrubbed logs, credentials no longer sent) is generally worth highlighting in a `.0` even though it fixes no reported bug.

## Auditing an existing highlight set

A `.0` diff range is large enough that the starting highlight set is usually too big and unevenly labeled. When trimming:

- Removing `impact/changelog` is sufficient to cut an entry. Leave the `area/*` label and the `release-note` block in place - the block is inert without `impact/changelog`, and retaining both preserves the wording if the entry is reinstated.
- Check whether cutting the last entry in a category empties a section (e.g. `#### ctr development tool`). An empty section disappears, which is usually correct, but confirm it is intended.
- Entries that predate the audit have not necessarily been held to any bar. Do not assume an existing `impact/changelog` label represents a considered decision.

## Additional verification

In addition to the checks in `SKILL.md`:

- The preface contains **no** `### Security Updates` section.
- Every entry intentionally cut during the audit is absent from the regenerated notes.
- No section was left empty, or unintentionally emptied, by the cuts.
