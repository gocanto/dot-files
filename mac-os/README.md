# mac-os

Go-based setup tooling for a new macOS machine.

This repo separates two jobs:

- Restore the practical setup: Homebrew packages, safe dotfiles, developer tools, and curated macOS defaults.
- Capture a private reference archive outside the repo for broad machine settings and inventory.

## Usage

For a new Mac, start from the repository root:

```sh
./setup.sh
```

The setup script requires macOS, network access, administrator/sudo access, and
completion of any Apple Command Line Tools installer or license prompts. It
installs or enables Xcode Command Line Tools, installs Homebrew and Go when
missing, then opens the Go TUI.

After bootstrap, or when developing this tool, open the TUI directly:

```sh
go run ./mac-os/cmd/mac-os
```

From inside this `mac-os` directory, the shorter module-local form also works:

```sh
go run ./cmd/mac-os
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

- `mac-os`: opens the Bubble Tea workflow dashboard.

Previous scriptable subcommands are no longer supported. Workflow execution now
runs through the TUI.

## Interactive TUI

`mac-os` uses Bubble Tea v2 (`charm.land/bubbletea/v2`) to present a terminal
dashboard for common workflows:

- Factory Install
- Factory Install Dry Run
- Bootstrap
- Capture Archive
- Restore App Configs
- Apply macOS Defaults
- Doctor
- Brewfile Preview

Use arrow keys or `j`/`k` to move, `enter` to open or run, `space` to toggle
phases in a workflow, and `q`/`esc` to exit before execution. The run screen
executes enabled phases in order, shows each phase status, stops on first
failure, and returns a non-zero exit code on failure or `ctrl+c` cancellation.

`Factory Install` is the one-pass setup path. It starts with an erase-state
confirmation, then installs prerequisites, Homebrew packages, App Store apps,
safe dotfiles, Stow links, macOS defaults, and doctor checks without walking
through each workflow separately.

`Factory Install Dry Run` exercises the same confirmation and phase progress in
a read-only mode. It prints what would happen, including the Erase Assistant
handoff, without opening reset settings or installing packages.

The current TUI defaults to dry-run-oriented workflows so it is safe as an
interactive front door.

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
gitconfig_plaintext    concealed private ~/.config/git/private.gitconfig contents
latest_archive         last encrypted archive path, updated by capture
restore_notes          short manual restore notes
```

Create the Age identity once and store the private identity in 1Password:

```sh
age-keygen
```

Copy the `AGE-SECRET-KEY-...` line into `archive_age_identity` and the
`age1...` public recipient into `archive_age_recipient`.

Use the TUI capture workflow to create a machine reference archive.

On a new Mac, run `./setup.sh` first. After Homebrew, 1Password, `op`,
and `age` are installed, sign in with `op`, retrieve the identity from
1Password, and decrypt the latest archive. Keep the identity file outside the
repo.

## Private Git config

Private dotfile overlays are declared in `secrets.yaml`. The tracked Stow Git
config includes `~/.config/git/private.gitconfig`, but the plaintext private
file is ignored by git. Store its contents in the `gitconfig_plaintext` field
above and keep the encrypted repo copy in:

```text
mac-os/stow/git/.config/git/private.gitconfig.age
```

The public CLI no longer exposes standalone secret-management subcommands.

## Developer tools

The Brewfile includes common CLI/dev tools, databases, AI tools, and app casks:

- Git/GitHub: `git`, `gh`.
- Shell/dev utilities: `jq`, `fd`, `fzf`, `glow`, `gnupg`, `vim`, `yazi`, `sevenzip`, `stow`.
- Private archive encryption: `age`.
- Runtimes/databases: `go`, `node@24`, `mysql`, `libpq`, `nginx`.
- AI/dev CLIs: `agent-browser`, `codex`, `claude-code`, `opencode`.
- Apps: Docker, VS Code, Ghostty, iTerm2, Raycast, Stats, 1Password, Bruno, Claude, Ice.

## Code organization

The Go module keeps `internal/app` as the CLI coordinator and splits behavior
into small packages:

- `internal/command`: command runner, shell quoting, 1Password field parsing.
- `internal/appconfig`: `apps.yaml` loading, validation, and app capture plans.
- `internal/safefs`: safe writes/copies, sensitive-path filtering, sanitization.
- `internal/dotfiles`: dotfile adopt and capture plans.
- `internal/macosdefaults`: curated defaults apply/export behavior.
- `internal/brewfile`: Brewfile generation.
- `internal/apps`: App Store install reporting and app config capture/restore.
- `internal/archive`: private archive capture, encryption, metadata updates.
- `internal/doctor`: prerequisite and developer tool checks.
- `internal/secrets`: encrypted private dotfile overlay workflows.
- `internal/tui`: Bubble Tea models, views, key handling, and phase execution.

## Safety

The capture flow is intentionally conservative. It skips private keys, shell histories, auth files, token-like paths, app caches, sessions, Claude/Codex file history, Docker VM data, and database data.

The defaults archive is for reference. The bootstrap does not raw-replay every exported defaults domain because macOS defaults can contain private values such as text replacements, addresses, account state, and machine-specific identifiers.
