# Mac Migration Manual

> A repeatable restore path for my Mac: tools, apps, safe dotfiles, curated
> macOS settings, and private recovery metadata without committing secrets.

This repository is the path back to a working Mac when I rebuild, replace, or
erase a machine. It keeps the setup I care about in one place so the restore is
based on deliberate policy instead of memory, scattered notes, or whatever
happens to still exist on the old laptop.

The goal is practical recovery. This repo installs the tools and apps I use,
restores safe dotfiles, applies known macOS defaults, captures reference
settings, and keeps private machine data out of git. Secrets, private keys,
browser sessions, keychains, database data, Docker VM data, and token-like files
are deliberately excluded from the repo and from automatic replay.

## Quick Links

| Need | Start here |
| --- | --- |
| Set up a fresh Mac | [`./setup.sh`](#fresh-mac-setup) |
| Run the terminal dashboard | [Using The TUI](#using-the-tui) |
| Understand what gets installed | [App Restore Policy](#app-restore-policy) |
| Restore private settings | [Private Archives](#private-archives) |
| Work on the tool | [Developer Notes](#developer-notes) |
| Know what is intentionally skipped | [Safety Rules](#safety-rules) |

## What This Repo Does

- Bootstraps a new macOS machine from the repository root with `./setup.sh`.
- Installs or enables Xcode Command Line Tools, Homebrew, Git, and Go when
  needed.
- Opens a terminal dashboard for setup, app restore, app snapshots, macOS
  defaults, health checks, and package review.
- Installs command-line tools, Homebrew casks, and App Store apps from tracked
  policy.
- Reports manual apps that still need human installation or login.
- Links safe shell, Git, Vim, and terminal configuration with Stow.
- Stores private archive metadata and selected secret text in 1Password instead
  of committing it.
- Captures supported app settings and machine reference files into private
  archives outside the repo.

> [!IMPORTANT]
> This is not a full Mac backup. It does not clone every app, account, browser
> profile, keychain, database, container, cache, or private credential. Use a
> real backup strategy for personal files and app data that must be preserved
> exactly.

## Fresh Mac Setup

Start from the repository root.

```sh
./setup.sh
```

### Requirements

| Requirement | Why it matters |
| --- | --- |
| macOS | The setup flow applies Mac-specific tools and defaults. |
| Network access | Homebrew, GitHub, 1Password, and package installs need it. |
| Administrator access | The bootstrap validates `sudo` and may install system tooling. |
| Xcode Command Line Tools prompts | Apple installer and license prompts must be completed before setup continues. |
| 1Password access | Private fields and archive metadata are restored from the configured vault. |

The script checks macOS, installs or enables Xcode Command Line Tools, installs
Homebrew and Git when missing, confirms it is running from the canonical git
checkout, starts a `sudo` keepalive, installs Go when missing, and then opens
the Go TUI.

If setup is launched from an unzipped download instead of a git checkout, it
prompts for a destination path, clones the repository there, and re-runs setup
from the clone. The default destination is:

```text
$HOME/Sites/dot-files
```

This matters because Stow links and later repo updates should point to one
stable checkout, not a temporary download directory.

## Using The TUI

After bootstrap, the setup flow runs through a Bubble Tea terminal dashboard.
You can also open it directly:

```sh
go run ./mac-os/cmd/mac-os
```

From inside the `mac-os` directory, this shorter form also works:

```sh
go run ./cmd/mac-os
```

### Controls

| Key | Action |
| --- | --- |
| Arrow keys or `j`/`k` | Move through choices. |
| `enter` | Open or run a workflow. |
| `space` | Toggle workflow phases. |
| `q` or `esc` | Exit before execution. |
| `ctrl+c` | Cancel a running workflow. |

Most workflows start with a confirmation screen that explains what will happen,
whether the workflow changes the Mac, and which phases will run. Workflows that
can change files or settings offer `Preview only` before `Run now`.

### Workflows

| Workflow | Purpose |
| --- | --- |
| `Set Up This Mac` | One-pass restore path for a new machine. |
| `Save App Settings Snapshot` | Captures supported app settings for review or later restore. |
| `Restore App Settings` | Restores allowlisted app settings from a private archive. |
| `Update Installed App List` | Scans installed apps and writes `mac-os/apps.generated.yaml` for review. |
| `Apply macOS Settings` | Applies curated macOS defaults. |
| `Check Setup` | Verifies prerequisites and developer tools. |
| `Show Homebrew Packages` | Prints the generated Homebrew package plan. |

`Set Up This Mac` installs prerequisites, Homebrew packages, GitHub access and
signing keys, App Store apps, private secrets from 1Password, safe dotfiles via
Stow, curated macOS settings, and health checks. If no active 1Password CLI
session exists, the GitHub and private secret phases prompt for `op signin`.

If you choose `Erase first`, the workflow validates administrator access, opens
Apple's Erase Assistant settings, and stops before install phases run.

## Dotfiles And Stow

Tracked dotfiles live under `mac-os/stow`. The setup workflow links safe shell,
Git, and Vim configuration into `$HOME` with Stow.

Before linking, the Stow phase scans `$HOME` for existing symlinks that point
into a different stow tree, such as an old run from `~/Downloads/dot-files-main`.
It refuses to continue until those links are removed. This avoids silent Stow
skips and keeps the machine linked to the canonical checkout.

Private Git configuration is handled separately. The tracked Git config includes
`~/.config/git/private.gitconfig`, but the plaintext file is ignored by git and
restored locally from 1Password.

## App Restore Policy

`mac-os/apps.yaml` is the source of truth for app install and restore behavior.
The `Update Installed App List` workflow never rewrites that file directly. It
scans installed GUI apps, Homebrew casks, and Mac App Store apps, then writes a
review candidate to `mac-os/apps.generated.yaml`.

### Install Methods

| Method | Behavior |
| --- | --- |
| `brew` | Installed by the generated Homebrew plan. |
| `mas` | Installed with the Mac App Store CLI. |
| `manual` | Reported with download or restore notes. |
| `system` | Expected to ship with macOS. |

### Config Modes

| Mode | Behavior |
| --- | --- |
| `auto` | Allowlisted paths can be captured and restored automatically. |
| `reference` | Paths are captured for review but not replayed. |
| `manual` | Restore depends on app sync, login, export/import, or manual notes. |

Even when a directory is allowlisted, the capture flow skips secrets, sessions,
browser profiles, keychains, SSH and GPG private keys, database data, Docker VM
data, histories, caches, and token-like files.

## Private Archives

Private snapshots are written outside the repo:

```text
$HOME/.local/state/macos-settings-archives/<timestamp>
```

Encrypted archives are written beside the timestamped capture directory:

```text
<timestamp>.tar.gz.age
```

The private archive workflow uses 1Password for secrets and metadata, not for
large archive storage. Keep encrypted archive files on an external drive or in
a cloud-synced encrypted folder outside this repository.

Default 1Password location:

```text
Vault: Private
Item:  Mac Migration Archive
```

### Expected 1Password Fields

| Field | Contents |
| --- | --- |
| `archive_age_identity` | Concealed Age private identity. |
| `archive_age_recipient` | Age public recipient. |
| `archive_root` | Directory for encrypted archives. |
| `gitconfig_plaintext` | Concealed private `~/.config/git/private.gitconfig` contents. |
| `allowed_signers_plaintext` | Concealed `~/.ssh/allowed_signers` contents. |
| `github_username` | GitHub username. |
| `github_email` | Git commit/GitHub email. |
| `git_author_name` | Git commit author name. |
| `latest_archive` | Last encrypted archive path, updated by capture. |
| `restore_notes` | Short manual restore notes. |

Create the Age identity once:

```sh
age-keygen
```

Store the `AGE-SECRET-KEY-...` line in `archive_age_identity` and the `age1...`
public recipient in `archive_age_recipient`. Keep any local identity file
outside the repo.

On a new Mac, run `./setup.sh` first. After Homebrew, 1Password, `op`, and
`age` are available, sign in with `op`, retrieve the identity from 1Password,
and decrypt the latest archive from the location recorded in 1Password.

## GitHub And Signing

The GitHub setup step reads these 1Password fields when available:

- `github_username`
- `github_email`
- `git_author_name`

If a value is missing, the workflow prompts for it. It creates machine-local SSH
and GPG keys, uploads only public keys to GitHub with `gh`, and writes the
resolved Git identity plus signing key to the private Git config. SSH and GPG
private keys are never committed and are not stored in 1Password by this repo.

## macOS Defaults

The setup flow applies curated macOS defaults, but it does not raw-replay every
exported defaults domain. Full defaults exports can contain private values such
as text replacements, addresses, account state, and machine-specific
identifiers, so broad defaults captures are kept as reference material instead
of blindly replayed.

## Developer Notes

Run the TUI directly from the repository root:

```sh
go run ./mac-os/cmd/mac-os
```

Run tests from the `mac-os` module:

```sh
cd mac-os
go test ./...
```

Format Go code from the repository root:

```sh
make format
```

`make format` runs the private `ghcr.io/oullin/go-fmt` image through
`go-fmt.compose.yaml`. Before the first pull, Docker needs a GHCR credential:

```sh
gh auth status
gh auth refresh -h github.com -s read:packages
make format-login
```

Re-run `make format-login` only when the Docker credential expires or is wiped.

## Safety Rules

- Keep private archives outside the repository.
- Keep Age identity files outside the repository.
- Keep private Git config plaintext ignored by git.
- Review captured app settings before restoring them to a different machine.
- Use the TUI preview mode when you want to inspect a workflow before changing
  the Mac.
- Treat this repo as setup automation, not as a substitute for full backups.

## License

See [LICENSE](LICENSE).
