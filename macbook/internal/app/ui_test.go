package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gocanto/mac-os/internal/workflowdomain"
)

func TestUIWorkflowsEmitsMetadata(t *testing.T) {
	var stdout bytes.Buffer
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{})

	if got := a.run([]string{"ui", "workflows"}); got != 0 {
		t.Fatalf("exit = %d, want 0", got)
	}

	var workflows []workflowdomain.WorkflowMetadata

	if err := json.Unmarshal(stdout.Bytes(), &workflows); err != nil {
		t.Fatal(err)
	}

	if len(workflows) != 8 || workflows[0].ID != "set-up-this-mac" {
		t.Fatalf("workflows = %#v", workflows)
	}
}

func TestUIRunPersistsEventsAndRunLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MAC_OS_UI_DB_PATH", filepath.Join(home, "runs.sqlite3"))

	request := workflowdomain.RunRequest{
		WorkflowID:           "show-homebrew-packages",
		ConfirmationOptionID: "run-now",
		EnabledPhaseIDs:      []string{"print-generated-homebrew-package-list"},
	}

	body, err := json.Marshal(request)

	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	a := newApp(home, "/repo", bytes.NewReader(body), &stdout, io.Discard, stubRunner{})

	if got := a.run([]string{"ui", "run"}); got != 0 {
		t.Fatalf("exit = %d stdout = %s", got, stdout.String())
	}

	runID := firstRunID(t, stdout.String())

	var runsOut bytes.Buffer

	a = newApp(home, "/repo", strings.NewReader(""), &runsOut, io.Discard, stubRunner{})

	if got := a.run([]string{"ui", "runs"}); got != 0 {
		t.Fatalf("runs exit = %d", got)
	}

	if !strings.Contains(runsOut.String(), "show-homebrew-packages") {
		t.Fatalf("runs output = %s", runsOut.String())
	}

	var logOut bytes.Buffer

	a = newApp(home, "/repo", strings.NewReader(""), &logOut, io.Discard, stubRunner{})

	if got := a.run([]string{"ui", "run-log", "--run-id", runID}); got != 0 {
		t.Fatalf("run-log exit = %d", got)
	}

	if !strings.Contains(logOut.String(), "phase_output") {
		t.Fatalf("run log = %s", logOut.String())
	}
}

func firstRunID(t *testing.T, output string) string {
	t.Helper()

	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		var event workflowdomain.Event

		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}

		if event.RunID != "" {
			return event.RunID
		}
	}

	t.Fatalf("no run id in output %q", output)

	return ""
}
