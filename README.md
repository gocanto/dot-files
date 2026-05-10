# gus-mac

gus-mac is a native macOS app for restoring and converging my Mac from
tracked policy. It applies developer tools, applications, Stow-managed dotfiles,
macOS defaults, selective snapshots, and private recovery metadata without
committing secrets to git.

The current app is an Electron/Vue desktop UI backed by Go workflows. The old
terminal UI is archived on the [`v1` branch](https://github.com/gocanto/dot-files/tree/v1).

## What This Is

This repository is the path back to a working Mac when a machine is rebuilt,
replaced, erased, or audited. It is not a full backup system. The app provides a
reviewable control surface for:

- inspecting the current Mac state;
- applying a tracked setup template;
- installing and removing managed apps;
- restoring supported app configuration from selective snapshots;
- keeping private values in 1Password or outside the repository.

The implementation lives in a small workspace:

| Path               | Role                                                       |
| ------------------ | ---------------------------------------------------------- |
| `packages/macbook` | Go CLI, local HTTP backend, workflows, services, templates. |
| `packages/ui`      | Native desktop app shell built with Electron and Vue.      |
| `packages/bridge`  | TypeScript client used by Electron to call the backend.    |
| `packages/tools`   | Turbo cache wrapper and unsigned macOS release helper.     |

## Who This Is For

This is primarily my personal Mac recovery and dotfiles system. It is tuned for
my app policy, 1Password fields, GitHub setup, shell config, and snapshot
allowlist.

It can still be useful as a reference if you want to build your own native Mac
setup app around a tracked source of truth, but it is not a generic bootstrapper
that should be run blindly on another machine.

## Why Care

A Mac rebuild usually mixes manual installs, scattered settings, private files,
and muscle memory. This app turns that into a repeatable workflow:

- tracked policy shows what should be installed and configured;
- preview steps make changes reviewable before they touch the machine;
- generated candidates separate current machine state from desired policy;
- snapshots recover selected app settings without becoming a full backup;
- secrets and private archives stay out of git.

## Start Here

For the full tutorial, workflow reference, safety rules, and release notes, read
[storage/docs/README.md](storage/docs/README.md).

Download the [latest release](https://github.com/gocanto/dot-files/releases/latest)
or review [all releases](https://github.com/gocanto/dot-files/releases).

From the repository root:

```sh
./setup.sh
pnpm install
pnpm --filter ui build
pnpm --filter ui start
```

For active development:

```sh
pnpm dev
```

## License

See [LICENSE](LICENSE).
