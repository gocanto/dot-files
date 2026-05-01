package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(name string, args ...string) ([]byte, error)
}

type RealRunner struct{}

func (RealRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)

	return cmd.CombinedOutput()
}

func ShellQuote(parts []string) string {
	quoted := make([]string, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			quoted = append(quoted, "''")

			continue
		}

		if strings.ContainsAny(part, " \t\n\"'\\$`!*?[]{}()&;|<>") {
			quoted = append(quoted, "'"+strings.ReplaceAll(part, "'", `'\''`)+"'")

			continue
		}

		quoted = append(quoted, part)
	}

	return strings.Join(quoted, " ")
}

func FirstLine(b []byte) string {
	line, _, _ := bytes.Cut(b, []byte("\n"))

	return string(line)
}

func OnePasswordFields(runner Runner, vault, item string) (map[string]string, error) {
	out, err := runner.Run("op", "item", "get", item, "--vault", vault, "--format", "json")

	if err != nil {
		return nil, fmt.Errorf("read 1Password item %q in vault %q: %w", item, vault, err)
	}

	var parsed struct {
		Fields []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Value string `json:"value"`
		} `json:"fields"`
	}

	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("parse 1Password item JSON: %w", err)
	}

	fields := make(map[string]string, len(parsed.Fields))

	for _, field := range parsed.Fields {
		if field.ID != "" {
			fields[field.ID] = field.Value
		}

		if field.Label != "" {
			fields[field.Label] = field.Value
		}
	}

	return fields, nil
}
