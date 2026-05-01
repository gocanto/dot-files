# mac-os

Go-based setup tooling for a new macOS machine.

This repo separates two jobs:

- Restore the practical setup: Homebrew packages, safe dotfiles, developer tools, and curated macOS defaults.
- Capture a private reference archive outside the repo for broad machine settings and inventory.

## Usage

For a new Mac, start from the repository root:

```sh
./setup.sh --apps
```

The setup script requires macOS, network access, administrator/sudo access, and
completion of any Apple Command Line Tools installer or license prompts. It
installs or enables Xcode Command Line Tools, installs Homebrew and Go when
missing, then runs the Go bootstrap flow.

After bootstrap, or when developing this tool, run commands directly:

```sh
go run ./mac-os/cmd/mac-os doctor
go run ./mac-os/cmd/mac-os bootstrap --apps --dry-run
go run ./mac-os/cmd/mac-os bootstrap --apps
go run ./mac-os/cmd/mac-os capture --apps --encrypt
go run ./mac-os/cmd/mac-os restore --archive "$HOME/.local/state/macos-settings-archives/<timestamp>" --apps --dry-run
```

From inside this `mac-os` directory, the shorter module-local form also works:

```sh
go run ./cmd/mac-os adopt --dry-run
go run ./cmd/mac-os doctor
go run ./cmd/mac-os bootstrap --apps --dry-run
go run ./cmd/mac-os capture --apps --encrypt
go run ./cmd/mac-os restore --archive "$HOME/.local/state/macos-settings-archives/<timestamp>" --apps --dry-run
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
- `bootstrap`: prompted phases for prerequisites, Homebrew, App Store apps, manual app reporting, safe dotfile adoption, Stow links, optional app config restore, macOS defaults, archive capture, and doctor checks.
- `capture`: writes a private machine inventory outside the repo; `--apps` adds allowlisted app configuration from `apps.yaml`.
- `restore`: restores only app configuration marked `auto` in `apps.yaml` from a private archive.
- `doctor`: prints required tools and developer tool versions.
- `brewfile`: prints or writes the curated Brewfile.
- `macos`: applies only the curated macOS defaults.

All mutating commands support `--dry-run`. Prompted commands support `--yes`.
App-aware commands accept `--config PATH`; by default they load `apps.yaml`
through Viper.
All `mac-os` commands validate sudo access at startup, including read-only
commands, because machine setup assumes administrator access.

## App restore policy

`apps.yaml` is the tracked source of truth for near-clone app restore behavior:

- `install_method: brew` apps are installed by the Brewfile.
- `install_method: mas` apps are installed with `mas install`.
- `install_method: manual` apps are reported with their download/source notes.
- `install_method: system` apps are expected to ship with macOS.
- `config_mode: auto` paths are captured and can be restored automatically.
- `config_mode: reference` paths are captured for review but not replayed.
- `config_mode: manual` requires app sync, login, export/import, or restore notes.

Secrets, sessions, browser profiles, keychains, SSH/GPG private keys, database
data, Docker VM data, and token-like files are intentionally skipped even when
they appear under an allowlisted directory.

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

Capture and encrypt the current machine, including allowlisted app config:

```sh
go run ./cmd/mac-os capture --apps --encrypt
```

Use a different vault, item, or archive location when needed:

```sh
go run ./cmd/mac-os capture --encrypt \
  --op-vault Private \
  --op-item "Mac Migration Archive" \
  --archive-root "/Volumes/Migration/macbook"
```

On a new Mac, run `./setup.sh --apps` first. After Homebrew, 1Password, `op`,
and `age` are installed, sign in with `op`, retrieve the identity from
1Password, and decrypt the latest archive. Keep the identity file outside the
repo.

## Developer tools

The Brewfile includes common CLI/dev tools, databases, AI tools, and app casks:

- Git/GitHub: `git`, `gh`.
- Shell/dev utilities: `jq`, `fd`, `fzf`, `glow`, `gnupg`, `vim`, `yazi`, `sevenzip`, `stow`.
- Private archive encryption: `age`.
- Runtimes/databases: `go`, `node@24`, `mysql`, `libpq`, `nginx`.
- AI/dev CLIs: `agent-browser`, `codex`, `claude-code`, `opencode`.
- Apps: Docker, VS Code, Ghostty, iTerm2, Raycast, Stats, 1Password, Bruno, Claude, Ice.

## Safety

The capture flow is intentionally conservative. It skips private keys, shell histories, auth files, token-like paths, app caches, sessions, Claude/Codex file history, Docker VM data, and database data.

The defaults archive is for reference. The bootstrap does not raw-replay every exported defaults domain because macOS defaults can contain private values such as text replacements, addresses, account state, and machine-specific identifiers.
