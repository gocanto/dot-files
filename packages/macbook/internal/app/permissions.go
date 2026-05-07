package app

import (
	"fmt"
	"io"
	"runtime"

	"github.com/gocanto/dot-files/internal/domain"
)

func (a app) approveHostPermissions(stdout io.Writer) error {
	goos := a.goos

	if goos == "" {
		goos = runtime.GOOS
	}

	if goos != "darwin" {
		return fmt.Errorf("host password approval is only supported on macOS, current OS is %s", goos)
	}

	fmt.Fprintln(stdout, "requesting host password approval on this Mac")

	script := `do shell script "/usr/bin/true" with prompt "Mac OS Manager needs your password to apply changes to this Mac." with administrator privileges`
	cmd := []string{"/usr/bin/osascript", "-e", script}

	if _, err := a.runner.Run(cmd[0], cmd[1:]...); err != nil {
		return fmt.Errorf("host password approval required: %w", err)
	}

	fmt.Fprintln(stdout, "host password approval accepted")

	return nil
}

func (a app) approvalOption(option domain.ConfirmationOption) domain.ConfirmationOption {
	option.RequiresApproval = true
	option.Approve = func(w io.Writer) error {
		return a.approveHostPermissions(w)
	}

	return option
}
