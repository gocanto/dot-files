package githubsetup

import (
	"fmt"
	"io"

	"github.com/gocanto/mac-os/internal/command"
)

type Options struct {
	DryRun  bool
	OPVault string
	OPItem  string
}

type Identity struct {
	GitHubUsername string
	GitHubEmail    string
	GitAuthorName  string
}

type Service struct {
	Home     string
	Repo     string
	Stdin    io.Reader
	Stdout   io.Writer
	Runner   command.Runner
	Hostname string
}

const (
	FieldGitHubUsername = "github_username"
	FieldGitHubEmail    = "github_email"
	FieldGitAuthorName  = "git_author_name"
)

func (s Service) Setup(opts Options) error {
	if err := s.ensureTools(opts.DryRun); err != nil {
		return err
	}

	identity, err := s.identity(opts)

	if err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would ensure GitHub CLI auth for %s\n", identity.GitHubUsername)
	} else if err := s.ensureGitHubAuth(); err != nil {
		return err
	}

	sshPub, err := s.ensureSSHKey(opts.DryRun, identity)

	if err != nil {
		return err
	}

	gpgKey, err := s.ensureGPGKey(opts.DryRun, identity)

	if err != nil {
		return err
	}

	if err := s.writePrivateGitconfig(opts.DryRun, identity, gpgKey); err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would upload SSH public key: %s\n", sshPub)
		fmt.Fprintf(s.Stdout, "would upload GPG public key: %s\n", gpgKey)

		return nil
	}

	if err := s.uploadSSHKey(sshPub); err != nil {
		return err
	}

	return s.uploadGPGKey(gpgKey)
}
