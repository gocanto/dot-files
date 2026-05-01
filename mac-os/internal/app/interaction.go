package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

func (a app) confirmAndRun(name string, opts options, fn func() error) error {
	fmt.Fprintf(a.stdout, "\n==> %s\n", name)

	if opts.dryRun {
		fmt.Fprintln(a.stdout, "dry-run mode: no changes will be applied")
	}

	if !opts.yes && !opts.dryRun {
		ok, err := a.confirm("Run this phase?")

		if err != nil {
			return err
		}

		if !ok {
			fmt.Fprintln(a.stdout, "skipped")

			return nil
		}
	}

	return fn()
}

func (a app) confirm(prompt string) (bool, error) {
	fmt.Fprintf(a.stdout, "%s [y/N] ", prompt)
	reader := bufio.NewReader(a.stdin)
	line, err := reader.ReadString('\n')

	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	answer := strings.ToLower(strings.TrimSpace(line))

	return answer == "y" || answer == "yes", nil
}

func (a app) requireSudo() error {
	out, err := a.runner.Run("sudo", "-v")

	if err != nil {
		message := strings.TrimSpace(string(out))

		if message != "" {
			return fmt.Errorf("run sudo -v: %w\n%s", err, message)
		}

		return fmt.Errorf("run sudo -v: %w", err)
	}

	return nil
}
