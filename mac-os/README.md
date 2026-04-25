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
```

Private archives are written to:

```text
$HOME/.local/state/macos-settings-archives/<timestamp>
```

## Commands

- `adopt`: imports safe current dotfiles into the repo's Stow layout and redacts known machine IDs.
- `bootstrap`: prompted phases for prerequisites, Homebrew, safe dotfile adoption, Stow links, macOS defaults, archive capture, and doctor checks.
- `capture`: writes a private machine inventory outside the repo.
- `doctor`: prints required tools and developer tool versions.
- `brewfile`: prints or writes the curated Brewfile.
- `macos`: applies only the curated macOS defaults.

All mutating commands support `--dry-run`. Prompted commands support `--yes`.

## Developer tools

The Brewfile includes common CLI/dev tools, databases, AI tools, and app casks:

- Git/GitHub: `git`, `gh`.
- Shell/dev utilities: `jq`, `fd`, `fzf`, `glow`, `gnupg`, `vim`, `yazi`, `sevenzip`, `stow`.
- Runtimes/databases: `node@24`, `mysql`, `libpq`, `nginx`.
- AI/dev CLIs: `agent-browser`, `codex`, `claude-code`, `opencode`.
- Apps: Docker, VS Code, Ghostty, iTerm2, Raycast, Stats, 1Password, Bruno, Claude, Ice.

## Safety

The capture flow is intentionally conservative. It skips private keys, shell histories, auth files, token-like paths, app caches, sessions, Claude/Codex file history, Docker VM data, and database data.

The defaults archive is for reference. The bootstrap does not raw-replay every exported defaults domain because macOS defaults can contain private values such as text replacements, addresses, account state, and machine-specific identifiers.
