package github

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
)

func (s Service) identity(opts Options) (Identity, error) {
	fields, err := command.OnePasswordFields(s.Runner, opts.OPVault, opts.OPItem)

	if err != nil {
		return Identity{}, err
	}

	identity := Identity{
		GitHubUsername: strings.TrimSpace(fields[FieldGitHubUsername]),
		GitHubEmail:    strings.TrimSpace(fields[FieldGitHubEmail]),
		GitAuthorName:  strings.TrimSpace(fields[FieldGitAuthorName]),
	}

	reader := bufio.NewReader(s.Stdin)

	if identity.GitHubUsername == "" {
		identity.GitHubUsername, err = promptRequired(reader, s.Stdout, "GitHub username")

		if err != nil {
			return Identity{}, err
		}
	}

	if identity.GitHubEmail == "" {
		identity.GitHubEmail, err = promptRequired(reader, s.Stdout, "GitHub email")

		if err != nil {
			return Identity{}, err
		}
	}

	if identity.GitAuthorName == "" {
		identity.GitAuthorName, err = promptRequired(reader, s.Stdout, "Git author name")

		if err != nil {
			return Identity{}, err
		}
	}

	return identity, nil
}

func promptRequired(reader *bufio.Reader, stdout io.Writer, label string) (string, error) {
	for {
		fmt.Fprintf(stdout, "%s: ", label)

		value, err := reader.ReadString('\n')

		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}

		value = strings.TrimSpace(value)

		if value != "" {
			return value, nil
		}

		fmt.Fprintf(stdout, "%s is required\n", label)

		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("%s is required", label)
		}
	}
}
