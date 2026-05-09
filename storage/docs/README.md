# macOS Manager

> A repeatable way to restore and converge a Mac from tracked policy: developer
> tools, applications, Stow-managed dotfiles, macOS defaults, selective
> snapshots, and private recovery metadata without committing secrets.

This repository is the path back to a working Mac when a machine is rebuilt,
replaced, erased, or audited. It is not a full backup. The app applies a tracked
template, captures reviewable state from the current Mac, restores supported app
configuration from local snapshots, and keeps private data in 1Password or
outside the repository.

The implementation is a Go workflow backend in `packages/macbook`, an
Electron/Vue desktop UI in `packages/ui`, a small TypeScript bridge package in
`packages/bridge`, and release/build helpers in `packages/tools`.

## Quick Start

Start from the repository root:

```sh
./setup.sh
```

The setup script supports fresh macOS machines. It checks macOS, installs or
enables Xcode Command Line Tools, installs Homebrew and Git when needed,
validates the checkout location, starts a `sudo` keepalive, installs Go when
missing, and checks that the Go workflow backend can start.

If setup is launched from an unzipped download instead of a git checkout, it
prompts for a canonical destination, clones the repository there, and re-runs
setup from the clone. The default destination is:

```text
$HOME/Sites/dot-files
```

Install the JavaScript workspace and launch the desktop app:

```sh
pnpm install
pnpm --filter ui build
pnpm --filter ui start
```

For active development, run the workspace dev command:

```sh
pnpm dev
```

The workflow backend can also be inspected from the terminal:

```sh
go run ./packages/macbook/cmd help
go run ./packages/macbook/cmd list-workflows
```

## How The App Works

The Electron app starts the Go backend with:

```text
api serve-http --socket <path>
```

The backend serves a local HTTP API over a per-launch Unix socket. The Electron
main process talks to it through `@dot-files/bridge`, and the UI presents the
same workflows that the CLI can list or run.

The UI is organized around these sections:

| Section    | Purpose                                                                 |
| ---------- | ----------------------------------------------------------------------- |
| `Source`   | Review tracked template files and generate review candidates.           |
| `This Mac` | Inspect installed tools, app inventory, current state, and snapshots.   |
| `Apply`    | Apply the template, restore snapshots, or remove untracked software.    |
| `Status`   | Show machine and prerequisite status.                                   |
| `Settings` | Configure repo paths, archive paths, workflow database, and 1Password.  |
| `Logs`     | Review persisted workflow runs and phase output.                        |

Most workflows open a confirmation screen before running. Workflows that may
write files or change the Mac provide `Preview only` before `Run now`.
Destructive workflows require explicit approval.

Runtime settings default to paths under `packages/macbook` and local state under
the user home directory. The repo root can be changed in the UI settings, and
relative runtime paths resolve from that root.

## Workflows

These are the current workflows exposed by the backend:

| Workflow                         | Purpose                                                                 | Side effects                                                              | When to use it                                                       |
| -------------------------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `Review Template`                | Validate and print the tracked source of truth.                         | Read-only.                                                                | After editing template files or before applying/removing anything.   |
| `Update Template From This Mac`  | Capture this Mac and write review candidates for template updates.      | Writes a snapshot, `apps.generated.yaml`, and dotfile candidates.          | After making manual changes that may become tracked policy.          |
| `Inspect Current State`          | Inspect this Mac without changing it.                                   | Read-only.                                                                | Before converging, removing apps, or debugging setup drift.          |
| `Regenerate Installed App List`  | Scan installed GUI, Homebrew, and Mac App Store apps.                   | Writes `packages/macbook/apps.generated.yaml`; never rewrites `apps.yaml`. | After app installs/removals or before updating app policy.           |
| `Save Snapshot`                  | Capture supported app settings and setup reference files.               | Writes a dated snapshot under the configured archive root.                 | Before risky changes, erasing a Mac, or preserving known-good state. |
| `Converge to Template`           | Apply the tracked template to this Mac.                                 | Installs tools/apps, restores secrets, links dotfiles, applies defaults.   | Fresh setup, re-converge after repo changes, or restore work.        |
| `Restore Snapshot`               | Restore supported app settings from a prior snapshot.                   | Overwrites targeted allowlisted app config files.                          | After reinstalling apps or recovering selected settings.             |
| `Remove Untracked Apps`          | Remove software not present in the tracked template.                    | Writes a pre-remove snapshot, uninstalls untracked Homebrew items.         | After auditing generated candidates and confirming desired policy.   |

`Converge to Template` has separate confirmation options:

| Option                      | Behavior                                                                 |
| --------------------------- | ------------------------------------------------------------------------ |
| `Preview only (fresh)`      | Shows the fresh setup plan without changing the Mac.                     |
| `Preview only (re-converge)` | Shows the re-converge plan without changing the Mac.                    |
| `Erase first`               | Opens Apple's reset assistant and stops before install phases continue.  |
| `Fresh setup`               | Installs policy and adopts existing dotfiles into the tracked Stow tree. |
| `Re-converge`               | Updates from policy and restores app configs from the latest snapshot.   |

The converge pipeline checks prerequisites, applies Homebrew packages, sets up
GitHub access and signing, installs App Store apps, reports manual app notes,
restores private secrets from 1Password, installs oh-my-zsh, links dotfiles,
applies macOS settings, and runs health checks.

## Template And State

Tracked runtime template files live under `packages/macbook`:

| Path                                      | Role                                                        |
| ----------------------------------------- | ----------------------------------------------------------- |
| `packages/macbook/apps.yaml`              | Source of truth for app install methods and restore policy. |
| `packages/macbook/apps.generated.yaml`    | Generated review candidate from installed app scans.        |
| `packages/macbook/secrets.yaml`           | References for private values restored from 1Password.      |
| `packages/macbook/stow/`                  | Safe shell, Git, Vim, and Ghostty files linked with Stow.   |
| `packages/macbook/internal/template/`     | Go template generators for Brewfile, dotfiles, secrets, and macOS defaults. |

`apps.yaml` is the app policy source of truth. Generated scans write
`apps.generated.yaml` for review; they do not overwrite policy.

App install methods:

| Method   | Behavior                                  |
| -------- | ----------------------------------------- |
| `brew`   | Installed by the generated Homebrew plan. |
| `mas`    | Installed with the Mac App Store CLI.     |
| `manual` | Reported with download or restore notes.  |
| `system` | Expected to ship with macOS.              |

App config modes:

| Mode        | Behavior                                                      |
| ----------- | ------------------------------------------------------------- |
| `auto`      | Allowlisted paths can be captured and restored automatically. |
| `reference` | Paths are captured for review but not replayed.               |
| `manual`    | Restore depends on sync, login, export/import, or notes.      |

Snapshots are selective archives, not full-machine backups. By default they are
written outside the repository:

```text
$HOME/.local/state/macos-settings-archives/<timestamp>
```

Encrypted archives are written beside the timestamped capture directory:

```text
<timestamp>.tar.gz.age
```

1Password stores secrets and archive metadata, not large archive files. The
default location is:

```text
Vault: Private
Item:  Mac Migration Archive
```

Expected 1Password fields:

| Field                       | Contents                                                      |
| --------------------------- | ------------------------------------------------------------- |
| `archive_age_identity`      | Concealed Age private identity.                               |
| `archive_age_recipient`     | Age public recipient.                                         |
| `archive_root`              | Directory for encrypted archives.                             |
| `gitconfig_plaintext`       | Concealed private `~/.config/git/private.gitconfig` contents. |
| `allowed_signers_plaintext` | Concealed `~/.ssh/allowed_signers` contents.                  |
| `github_username`           | GitHub username.                                              |
| `github_email`              | Git commit/GitHub email.                                      |
| `git_author_name`           | Git commit author name.                                       |
| `latest_archive`            | Last encrypted archive path, updated by capture.              |
| `restore_notes`             | Short manual restore notes.                                   |

Create the Age identity once:

```sh
age-keygen
```

Store the `AGE-SECRET-KEY-...` line in `archive_age_identity` and the `age1...`
public recipient in `archive_age_recipient`. Keep any local identity file
outside the repo.

Private Git configuration is restored separately. The tracked Git config
includes `~/.config/git/private.gitconfig`, but the plaintext file is ignored by
git and restored locally from 1Password.

## Safety Rules

- Treat this repository as setup and convergence automation, not a replacement
  for Time Machine, cloud sync, or app-native backups.
- Keep private archives, Age identity files, browser profiles, keychains,
  private SSH/GPG keys, databases, Docker VM data, caches, histories, sessions,
  and token-like files outside git.
- Use workflow preview before changing a machine when the expected file or app
  changes are not obvious.
- Review `packages/macbook/apps.generated.yaml` and dotfile candidates before
  moving generated state into tracked policy.
- Review snapshots before restoring them to a different machine.
- `Remove Untracked Apps` is destructive. It writes a pre-remove snapshot first
  and requires approval, but the tracked template must already represent the
  desired software state.
- `Erase first` only opens Apple's reset assistant and stops; it does not
  continue install phases after launching erase settings.
- The app applies curated macOS defaults instead of raw-replaying every exported
  defaults domain, because full exports may contain private or machine-specific
  values.

## Developer Notes

Workspace layout:

| Path                 | Role                                                       |
| -------------------- | ---------------------------------------------------------- |
| `packages/macbook`   | Go CLI, local HTTP backend, workflows, services, templates. |
| `packages/ui`        | Electron/Vue desktop UI.                                  |
| `packages/bridge`    | TypeScript client used by Electron to call the backend.    |
| `packages/tools`     | Turbo cache wrapper and unsigned macOS release helper.     |

Install dependencies:

```sh
pnpm install
```

Common checks from the repository root:

```sh
pnpm check
pnpm --dir packages/tools run turbo -- run build
pnpm --dir packages/tools run turbo -- run test
```

Build and open the Electron UI:

```sh
pnpm --filter ui build
pnpm --filter ui start
```

Regenerate the native macOS app icon from the tracked SVG source:

```sh
pnpm --filter ui run icon:generate
```

Run Go tests directly:

```sh
go test ./packages/macbook/...
```

The root `go test ./...` pattern is not used because `go.work` only includes
the `./packages/macbook` module.

Format code:

```sh
make format
```

`make format` runs the private `ghcr.io/oullin/go-fmt` image through
`go-fmt.compose.yaml`, then runs `oxfmt` and `oxlint`. Before the first pull,
Docker needs a GHCR credential:

```sh
gh auth status
gh auth refresh -h github.com -s read:packages
make format-login
```

Build an unsigned macOS DMG and ZIP while Developer ID approval is pending:

```sh
pnpm --dir packages/macbook run build
pnpm --dir packages/ui run build
pnpm --dir packages/ui run dist:mac:unsigned
```

Create a published GitHub release:

```sh
pnpm release:mac:unsigned -- --notes-file /path/to/release-notes.md --repo gocanto/dot-files
```

Unsigned builds require a manual first launch. Use right-click -> Open, or
remove quarantine manually:

```sh
xattr -dr com.apple.quarantine "/Applications/macOS Manager.app"
```

After Developer ID approval, use `pnpm --dir packages/ui run dist:mac:signed`
with Apple signing/notarization credentials configured for Electron Builder.
Enable auto-update only after signed and notarized artifacts pass Gatekeeper
checks.

## License

See [LICENSE](../../LICENSE).
