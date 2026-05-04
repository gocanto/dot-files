package doctor

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
)

func (s Service) EnsurePrerequisites(dryRun bool) error {
	goos := s.GOOS

	if goos == "" {
		goos = runtime.GOOS
	}

	if goos != "darwin" {
		return fmt.Errorf("mac-os only supports darwin, current OS is %s", goos)
	}

	cmd := []string{"xcode-select", "-p"}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))
		fmt.Fprintln(s.Stdout, "would check Xcode Command Line Tools license status")

		return s.ensureAppleSiliconSupport(dryRun)
	}

	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if err != nil {
		return fmt.Errorf("Xcode Command Line Tools are missing or unusable; run `xcode-select --install`, complete Apple's installer, then rerun setup\n%s", strings.TrimSpace(string(out)))
	}

	fmt.Fprintf(s.Stdout, "%s ok\n", cmd[0])

	if out, err := s.Runner.Run("xcodebuild", "-license", "check"); err != nil {
		message := strings.TrimSpace(string(out))
		lower := strings.ToLower(message)

		if strings.Contains(lower, "license") || strings.Contains(lower, "agree") {
			return fmt.Errorf("Xcode Command Line Tools license needs attention; run `sudo xcodebuild -license` and accept Apple's prompts\n%s", message)
		}
	}

	return s.ensureAppleSiliconSupport(dryRun)
}
