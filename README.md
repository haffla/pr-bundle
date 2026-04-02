# merge-prs

Batch-merge multiple GitHub PRs into a single branch. Useful for combining dependabot PRs and deploying them together to a staging environment.

## Installation

### Download a binary

Grab the latest release for your platform from the [releases page](https://github.com/haffla/merge-prs/releases).

### Install with Go

```
go install github.com/haffla/merge-prs@latest
```

### Prerequisites

- [GitHub CLI](https://cli.github.com) (`gh`) — must be authenticated via `gh auth login`

## Usage

```
merge-prs [OPTIONS] [PR_NUMBER ...]
```

### Merge specific PRs

```
merge-prs 721 720 719
```

### Interactive mode

Run without arguments to get a multi-select picker of open dependabot PRs:

```
merge-prs
```

Use `--all` to include PRs from all authors:

```
merge-prs --all
```

### Options

```
--branch NAME    Custom branch name (default: merge-prs/<timestamp>)
--all            Show all open PRs in interactive mode (default: dependabot only)
--target BRANCH  Remote branch for push prompt (default: staging)
```

### Conflict handling

If a merge produces conflicts, the script pauses and lets you resolve them in another terminal. You then type:

- `continue` — stage resolved files and carry on
- `skip` — skip that PR and move to the next
- `abort` — stop merging entirely

After all merges complete, you're prompted whether to push the result to a remote branch (defaults to `staging`).
