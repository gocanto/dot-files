package github

import (
	"fmt"

	"github.com/gocanto/mac-os/internal/command"
)

func (s Service) ensureGitHubAuth() error {
	if _, err := s.Runner.Run("gh", "auth", "status"); err == nil {
		fmt.Fprintln(s.Stdout, "GitHub CLI auth found")

		return nil
	}

	fmt.Fprintln(s.Stdout, "GitHub CLI is not authenticated; running: gh auth login")

	if err := command.RunInteractive(s.Runner, s.Stdout, "gh", "auth", "login"); err != nil {
		return fmt.Errorf("gh auth login failed: %w", err)
	}

	return nil
}
