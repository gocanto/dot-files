package app

import "fmt"

func (a app) usage() {
	fmt.Fprintln(a.stdout, `mac-os manages this machine's dotfiles, developer tools, and macOS settings.

Usage:
  mac-os
  mac-os tui
  mac-os adopt [--dry-run] [--yes]
  mac-os bootstrap [--archive PATH] [--apps] [--config PATH] [--dry-run] [--yes]
  mac-os capture [--apps] [--config PATH] [--archive-root PATH] [--encrypt] [--op-vault VAULT] [--op-item ITEM] [--dry-run] [--yes]
  mac-os restore --archive PATH [--apps] [--config PATH] [--dry-run] [--yes]
  mac-os secrets encrypt [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets decrypt [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets sync [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os doctor
  mac-os brewfile [--write PATH]
  mac-os macos [--dry-run] [--yes]

Commands:
  tui        Open the interactive Bubble Tea workflow dashboard.
  adopt      Import safe current dotfiles into the repo's Stow layout.
  bootstrap  Run prompted phases for tools, dotfiles, macOS defaults, capture, and doctor.
  capture    Save a private settings inventory outside the repo by default.
  restore    Restore allowlisted app configuration from a private archive.
  secrets    Manage encrypted private dotfile overlays with 1Password and Age.
  doctor     Print installed tool versions and missing prerequisites.
  brewfile   Print or write the curated Brewfile for this setup.
  macos      Apply curated macOS defaults only.`)
}

func (a app) secretsUsage() {
	fmt.Fprintln(a.stdout, `Usage:
  mac-os secrets encrypt [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets decrypt [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets sync [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets encrypt-gitconfig [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets decrypt-gitconfig [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets sync-gitconfig [--op-vault VAULT] [--op-item ITEM] [--dry-run]`)
}
