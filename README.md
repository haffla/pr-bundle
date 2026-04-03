# pr-bundle

Batch-merge multiple GitHub PRs into a single branch. Useful for combining dependabot PRs and deploying them together to a staging environment.

## Installation

### Quick install

```sh
# macOS Apple Silicon
gh release download --repo haffla/pr-bundle --pattern 'pr-bundle_darwin_arm64*' --output - | tar xz

# macOS Intel
gh release download --repo haffla/pr-bundle --pattern 'pr-bundle_darwin_amd64*' --output - | tar xz

# Linux x86_64
gh release download --repo haffla/pr-bundle --pattern 'pr-bundle_linux_amd64*' --output - | tar xz

# Linux ARM
gh release download --repo haffla/pr-bundle --pattern 'pr-bundle_linux_arm64*' --output - | tar xz

# Then move it onto your PATH
chmod +x pr-bundle
sudo mv pr-bundle /usr/local/bin/
```

### Install with Go

```
go install github.com/haffla/pr-bundle@latest
```

### Prerequisites

- [GitHub CLI](https://cli.github.com) (`gh`) — must be authenticated via `gh auth login`

## Usage

```
pr-bundle [OPTIONS] [PR_NUMBER ...]
```

### Merge specific PRs

```
pr-bundle 721 720 719
```

### Interactive mode

Run without arguments to get a multi-select picker of open dependabot PRs:

```
pr-bundle
```

Use `--all` to include PRs from all authors:

```
pr-bundle --all
```

### Options

```
--branch NAME    Custom branch name (default: pr-bundle/<timestamp>)
--all            Show all open PRs in interactive mode (default: dependabot only)
--target BRANCH  Remote branch for push prompt (default: staging)
```

### Conflict handling

If a merge produces conflicts, the script pauses and lets you resolve them in another terminal. You then type:

- `continue` — stage resolved files and carry on
- `skip` — skip that PR and move to the next
- `abort` — stop merging entirely

After all merges complete, you're prompted whether to push the result to a remote branch (defaults to `staging`).
