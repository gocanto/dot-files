#!/bin/zsh

set -euo pipefail

DRY_RUN=0

for arg in "$@"; do
	case "$arg" in
		--dry-run)
			DRY_RUN=1
			;;
	esac
done

log() {
	printf '==> %s\n' "$1"
}

die() {
	printf 'error: %s\n' "$1" >&2
	exit 1
}

run() {
	if [[ "$DRY_RUN" -eq 1 ]]; then
		printf 'would run:'
		printf ' %q' "$@"
		printf '\n'
		return 0
	fi

	"$@"
}

start_sudo_keepalive() {
	if [[ "$DRY_RUN" -eq 1 ]]; then
		log "would validate sudo access with sudo -v"
		return 0
	fi

	log "validating sudo access"
	sudo -v || die "administrator access is required"

	while true; do
		sudo -n -v
		sleep 60
	done 2>/dev/null &

	SUDO_KEEPALIVE_PID=$!
	trap 'kill "$SUDO_KEEPALIVE_PID" 2>/dev/null || true' EXIT INT TERM
}

ensure_macos() {
	if [[ "$(uname -s)" != "Darwin" ]]; then
		die "setup only supports macOS"
	fi
}

ensure_command_line_tools() {
	if xcode-select -p >/dev/null 2>&1; then
		log "Xcode Command Line Tools found"
	else
		log "Xcode Command Line Tools are missing"

		if [[ "$DRY_RUN" -eq 1 ]]; then
			log "would open Apple's Command Line Tools installer with xcode-select --install"
			log "would wait until xcode-select -p succeeds"
			return 0
		fi

		xcode-select --install 2>/dev/null || true
		printf 'Complete the Apple Command Line Tools installer dialog to continue.\n'

		until xcode-select -p >/dev/null 2>&1; do
			sleep 10
			printf '.'
		done

		printf '\n'
		log "Xcode Command Line Tools installed"
	fi

	if [[ "$DRY_RUN" -eq 1 ]]; then
		log "would check Xcode Command Line Tools license status"
		return 0
	fi

	if ! xcodebuild -license check >/tmp/dot-files-xcode-license.log 2>&1; then
		if grep -qiE 'license|agree' /tmp/dot-files-xcode-license.log; then
			cat /tmp/dot-files-xcode-license.log >&2
			die "Xcode license needs attention; run 'sudo xcodebuild -license' and accept Apple's prompts, then rerun setup"
		fi
	fi
}

load_homebrew() {
	local brew_bin=""

	if command -v brew >/dev/null 2>&1; then
		brew_bin="$(command -v brew)"
	elif [[ -x /opt/homebrew/bin/brew ]]; then
		brew_bin="/opt/homebrew/bin/brew"
	elif [[ -x /usr/local/bin/brew ]]; then
		brew_bin="/usr/local/bin/brew"
	fi

	if [[ -n "$brew_bin" ]]; then
		eval "$("$brew_bin" shellenv)"
	fi
}

ensure_homebrew() {
	load_homebrew

	if command -v brew >/dev/null 2>&1; then
		log "Homebrew found"
		return 0
	fi

	log "Homebrew is missing"

	if [[ "$DRY_RUN" -eq 1 ]]; then
		log "would install Homebrew with the official install script"
		return 0
	fi

	/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
	load_homebrew

	command -v brew >/dev/null 2>&1 || die "Homebrew installed but brew is not on PATH"
}

ensure_go() {
	if command -v go >/dev/null 2>&1; then
		log "Go found"
		return 0
	fi

	log "Go is missing"
	run brew install go
}

run_bootstrap() {
	local root
	root="$(cd "$(dirname "$0")" && pwd)"

	log "running mac-os bootstrap"

	if [[ "$DRY_RUN" -eq 1 ]]; then
		printf 'would run: go run ./mac-os/cmd/mac-os bootstrap --apps'
		printf ' %q' "$@"
		printf '\n'
		return 0
	fi

	cd "$root"
	go run ./mac-os/cmd/mac-os bootstrap --apps "$@"
}

ensure_macos
start_sudo_keepalive
ensure_command_line_tools
ensure_homebrew
ensure_go
run_bootstrap "$@"
