package brew

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gocanto/dot-files/internal/command"
	"github.com/gocanto/dot-files/internal/lineparse"
)

type Kind string

type Service struct {
	Stdout io.Writer
	Runner command.Runner
}

const (
	KindFormula Kind = "formula"
	KindCask    Kind = "cask"
)

func (s Service) InstalledFormulae() ([]string, error) {
	out, err := s.Runner.Run("brew", "list", "--formula", "-1")

	if err != nil {
		return nil, fmt.Errorf("brew list --formula: %w", err)
	}

	return parseList(out), nil
}

func (s Service) InstalledCasks() ([]string, error) {
	out, err := s.Runner.Run("brew", "list", "--cask", "-1")

	if err != nil {
		return nil, fmt.Errorf("brew list --cask: %w", err)
	}

	return parseList(out), nil
}

func Untracked(installed, tracked []string) []string {
	trackedSet := make(map[string]struct{}, len(tracked))

	for _, name := range tracked {
		trackedSet[strings.TrimSpace(name)] = struct{}{}
	}

	out := make([]string, 0)

	for _, name := range installed {
		name = strings.TrimSpace(name)

		if name == "" {
			continue
		}

		if _, ok := trackedSet[name]; ok {
			continue
		}

		out = append(out, name)
	}

	sort.Strings(out)

	return out
}

func (s Service) Uninstall(kind Kind, name string, dryRun bool) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("uninstall: empty name")
	}

	cmd := []string{"brew", "uninstall", "--zap", name}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would run: %s # %s\n", command.ShellQuote(cmd), kind)

		return nil
	}

	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("brew uninstall %s %q: %w", kind, name, err)
	}

	return nil
}

func parseList(out []byte) []string {
	return lineparse.UniqueTrimmed(out)
}
