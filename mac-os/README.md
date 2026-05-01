# mac-os

Go-based setup tooling for a new macOS machine.

This repo separates two jobs:

- Restore the practical setup: Homebrew packages, safe dotfiles, developer tools, and curated macOS defaults.
- Capture a private reference archive outside the repo for broad machine settings and inventory.

## Usage

```sh
go run ./cmd/mac-os adopt --dry-run
go run ./cmd/mac-os doctor
go run ./cmd/mac-os bootstrap --dry-run
go run ./cmd/mac-os bootstrap
go run ./cmd/mac-os capture
go run ./cmd/mac-os capture --encrypt
```

Private archives are written to:

```text
$HOME/.local/state/macos-settings-archives/<timestamp>
```

Encrypted archives are written beside the timestamped capture directory as:

```text
<timestamp>.tar.gz.age
```

## Commands

- `adopt`: imports safe current dotfiles into the repo's Stow layout and redacts known machine IDs.
- `bootstrap`: prompted phases for prerequisites, Homebrew, safe dotfile adoption, Stow links, macOS defaults, archive capture, and doctor checks.
- `capture`: writes a private machine inventory outside the repo.
- `doctor`: prints required tools and developer tool versions.
- `brewfile`: prints or writes the curated Brewfile.
- `macos`: applies only the curated macOS defaults.

All mutating commands support `--dry-run`. Prompted commands support `--yes`.

## 1Password private archive

The private archive workflow uses 1Password for secrets and metadata, not for
large archive files. The encrypted archive itself should live on an external
drive or in a cloud-synced encrypted folder outside this repo.

Default 1Password location:

```text
Vault: Private
Item:  Mac Migration Archive
```

Expected fields:

```text
archive_age_identity   concealed Age private identity
archive_age_recipient  Age public recipient
archive_root           directory for encrypted archives
latest_archive         last encrypted archive path, updated by capture
restore_notes          short manual restore notes
```

Create the Age identity once and store the private identity in 1Password:

```sh
age-keygen
```

Copy the `AGE-SECRET-KEY-...` line into `archive_age_identity` and the
`age1...` public recipient into `archive_age_recipient`.

Capture and encrypt the current machine:

```sh
go run ./cmd/mac-os capture --encrypt
```

Use a different vault, item, or archive location when needed:

```sh
go run ./cmd/mac-os capture --encrypt \
  --op-vault Private \
  --op-item "Mac Migration Archive" \
  --archive-root "/Volumes/Migration/macbook"
```

On a new Mac, install Homebrew and 1Password first, sign in with `op`, install
`age`, then retrieve the identity from 1Password and decrypt the latest archive.
Keep the identity file outside the repo.

## Developer tools

The Brewfile includes common CLI/dev tools, databases, AI tools, and app casks:

- Git/GitHub: `git`, `gh`.
- Shell/dev utilities: `jq`, `fd`, `fzf`, `glow`, `gnupg`, `vim`, `yazi`, `sevenzip`, `stow`.
- Private archive encryption: `age`.
- Runtimes/databases: `node@24`, `mysql`, `libpq`, `nginx`.
- AI/dev CLIs: `agent-browser`, `codex`, `claude-code`, `opencode`.
- Apps: Docker, VS Code, Ghostty, iTerm2, Raycast, Stats, 1Password, Bruno, Claude, Ice.

## Safety

The capture flow is intentionally conservative. It skips private keys, shell histories, auth files, token-like paths, app caches, sessions, Claude/Codex file history, Docker VM data, and database data.

The defaults archive is for reference. The bootstrap does not raw-replay every exported defaults domain because macOS defaults can contain private values such as text replacements, addresses, account state, and machine-specific identifiers.
