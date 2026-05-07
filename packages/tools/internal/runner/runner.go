package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	LookPath(file string) error
	Run(ctx context.Context, spec CommandSpec) error
	Output(ctx context.Context, spec CommandSpec) (string, error)
}

type CommandSpec struct {
	Cwd    string
	Name   string
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

type ExecRunner struct{}

type ExitError struct {
	Name     string
	Args     []string
	ExitCode int
	Err      error
}

func (ExecRunner) LookPath(file string) error {
	if _, err := exec.LookPath(file); err != nil {
		return fmt.Errorf("required command not found: %s", file)
	}

	return nil
}

func (ExecRunner) Run(ctx context.Context, spec CommandSpec) error {
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	cmd.Dir = spec.Cwd
	cmd.Stdin = chooseReader(spec.Stdin, os.Stdin)
	cmd.Stdout = chooseWriter(spec.Stdout, os.Stdout)
	cmd.Stderr = chooseWriter(spec.Stderr, os.Stderr)

	if err := cmd.Run(); err != nil {
		return commandError(spec, err)
	}

	return nil
}

func (ExecRunner) Output(ctx context.Context, spec CommandSpec) (string, error) {
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	cmd.Dir = spec.Cwd

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	out, err := cmd.Output()

	if err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return "", fmt.Errorf("%s %s: %s: %w", spec.Name, strings.Join(spec.Args, " "), message, err)
		}

		return "", commandError(spec, err)
	}

	return strings.TrimSpace(string(out)), nil
}

func commandError(spec CommandSpec, err error) error {

	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return &ExitError{
			Name:     spec.Name,
			Args:     append([]string(nil), spec.Args...),
			ExitCode: exitErr.ExitCode(),
			Err:      err,
		}
	}

	return fmt.Errorf("%s %s: %w", spec.Name, strings.Join(spec.Args, " "), err)
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("%s %s exited with status %d", e.Name, strings.Join(e.Args, " "), e.ExitCode)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

func chooseWriter(preferred io.Writer, fallback io.Writer) io.Writer {
	if preferred != nil {
		return preferred
	}

	return fallback
}

func chooseReader(preferred io.Reader, fallback io.Reader) io.Reader {
	if preferred != nil {
		return preferred
	}

	return fallback
}
