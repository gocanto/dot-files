package doctor

import (
	"io"

	"github.com/gocanto/mac-os/internal/command"
)

type Tool struct {
	Name        string
	VersionArgs []string
}

type Service struct {
	GOOS         string
	GOARCH       string
	Home         string
	Repo         string
	Stdout       io.Writer
	Runner       command.Runner
	rosettaCheck func() bool
}

const rosettaMarker = "/Library/Apple/usr/share/rosetta/rosetta"
const dockerDesktopSettingsPath = "Library/Group Containers/group.com.docker/settings-store.json"

func DevTools() []Tool {
	return []Tool{
		{"git", []string{"--version"}},
		{"gh", []string{"--version"}},
		{"node", []string{"--version"}},
		{"npm", []string{"--version"}},
		{"pnpm", []string{"--version"}},
		{"yarn", []string{"--version"}},
		{"python3", []string{"--version"}},
		{"go", []string{"version"}},
		{"php", []string{"--version"}},
		{"composer", []string{"--version"}},
		{"mas", []string{"version"}},
		{"mysql", []string{"--version"}},
		{"psql", []string{"--version"}},
		{"docker", []string{"--version"}},
		{"claude", []string{"--version"}},
		{"codex", []string{"--version"}},
		{"opencode", []string{"--version"}},
		{"agent-browser", []string{"--version"}},
		{"op", []string{"--version"}},
		{"age", []string{"--version"}},
	}
}
