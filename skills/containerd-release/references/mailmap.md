# containerd Mailmap Guide

This guide describes how to maintain the `.mailmap` file to ensure that contributors' human names are correctly represented in the release notes.

## When to Update Mailmap
Update the `.mailmap` file if the `release-tool` output:
1.  **Duplicate Contributors**: Shows the same person multiple times with different email addresses or slightly different names.
2.  **Usernames Only**: Shows a contributor by their GitHub username instead of their human name.
3.  **Multiple Names**: Warns that a contributor has "multiple names" or "multiple emails."

## Finding Correct Information
Use `git log` to search for different identities associated with a contributor.

### Search by Email:
```bash
git log --all --format="%aN <%aE>" | grep -i "<email>" | sort -u
```

### Search by Name:
```bash
git log --all --format="%aN <%aE>" | grep -i "<name>" | sort -u
```

### Check Other Branches:
The `main` branch usually has the most up-to-date `.mailmap`.
```bash
git show main:.mailmap | grep -i "<identifier>"
```

## Adding Entries to .mailmap
The format for `.mailmap` entries is:
`Proper Name <proper@email.com> [Commit Name] [<commit@email.com>]`

### Examples:

**1. Consolidate different emails under a single name:**
```
Real Name <preferred@email.com> <other@email.com>
```

**2. Map a username/email to a proper name:**
```
Real Name <preferred@email.com> Username <preferred@email.com>
```

**3. Fix a name typo for a specific email:**
```
Real Name <email@address.com> Typo Name <email@address.com>
```

## Consistency Rules
- **Prefer Human Names**: If a human name is found in any branch or commit, use it.
- **Project Consistency**: If a contributor is consistently represented as a username across the project, it is acceptable to leave it.
- **No Manual Searches**: Do not perform external searches (e.g., Google or GitHub profile lookups) unless explicitly requested. Rely on information already present in the repository and git history.
