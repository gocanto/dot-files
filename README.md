# Mac migration files

Personal setup and migration tooling for rebuilding a MacBook.

This started as dotfiles, but the repo now treats dotfiles as one part of a
larger restore flow:

- bootstrap a fresh Mac from admin access, installing/enabling Xcode Command
  Line Tools, Homebrew, and Go as needed;
- install command-line tools and apps with Homebrew;
- install App Store apps with `mas` and report manual app installs from a
  tracked Viper-loaded app policy;
- link safe shell, git, Vim, and terminal config with Stow;
- apply curated macOS defaults;
- capture a private machine inventory and allowlisted app config outside the
  repo;
- use 1Password to store private archive metadata and encryption keys.

See [mac-os/README.md](mac-os/README.md) for the restore workflow.


### License

Please see the [license file](https://github.com/gocanto/dot-files/blob/main/LICENSE) for more information.

## How can I thank you?

- :arrow_up: Follow me on [Twitter](https://twitter.com/gocanto).
- :star: Star the repository.
- :handshake: Open a pull request to fix/improve the codebase.
- :writing_hand: Open a pull request to improve the documentation.
- :email: Let's connect in [LinkedIn](https://www.linkedin.com/in/gocanto/).

> Thank you for reading this far. :blush:
