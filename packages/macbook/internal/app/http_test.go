package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gocanto/mac-os/internal/storage"
)

type sseEvent struct {
	Event string
	Data  string
}

func TestHTTPListWorkflowsReturnsMetadata(t *testing.T) {
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	server := httptest.NewServer(httpServer{app: a}.buildMux())

	defer server.Close()

	body := getJSON(t, server.URL+"/v1/workflows")
	workflows, _ := body["workflows"].([]any)
	ids := map[string]bool{}

	for _, row := range workflows {
		item, _ := row.(map[string]any)
		id, _ := item["id"].(string)
		ids[id] = true
	}

	for _, id := range []string{"review-template", "update-template-from-this-mac"} {
		if !ids[id] {
			t.Fatalf("missing workflow %q in %#v", id, workflows)
		}
	}
}

func TestHTTPRunPersistsEventsAndRunLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MAC_OS_WORKFLOW_DB_PATH", filepath.Join(home, "runs.sqlite3"))

	a := newApp(home, "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	server := httptest.NewServer(httpServer{app: a}.buildMux())

	defer server.Close()

	events := streamSSE(t, server.URL+"/v1/workflows/run", `{
		"workflowId": "review-template",
		"confirmationOptionId": "run-now",
		"enabledPhaseIds": ["print-tracked-homebrew-bundle"]
	}`)

	runID := firstSSERunID(t, events)

	if !hasSSEEventType(events, "phase_output") {
		t.Fatalf("expected phase_output event in stream, got %#v", events)
	}

	workflowEvents := workflowSSEEvents(t, events)
	outputIndex := eventTypeIndex(workflowEvents, "phase_output")
	finishIndex := eventTypeIndex(workflowEvents, "phase_finished")

	if outputIndex < 0 || finishIndex < 0 || outputIndex > finishIndex {
		t.Fatalf("expected phase_output before phase_finished, got %#v", workflowEvents)
	}

	runs := getJSON(t, server.URL+"/v1/runs")
	rows, _ := runs["runs"].([]any)

	if len(rows) != 1 {
		t.Fatalf("runs = %#v", rows)
	}

	if first, _ := rows[0].(map[string]any); first["workflowId"] != "review-template" {
		t.Fatalf("run = %#v", first)
	}

	log := getJSON(t, server.URL+"/v1/runs/"+runID+"/log")
	logEvents, _ := log["events"].([]any)

	if len(logEvents) == 0 {
		t.Fatalf("run log = %#v", log)
	}

	hasPhaseOutput := false

	for _, event := range logEvents {
		entry, _ := event.(map[string]any)

		if entry["type"] == "phase_output" {
			hasPhaseOutput = true

			break
		}
	}

	if !hasPhaseOutput {
		t.Fatalf("run log missing phase_output: %#v", logEvents)
	}
}

func TestHTTPSettingsValidation(t *testing.T) {
	home := t.TempDir()
	repo := writeSettingsRepo(t)
	a := newApp(home, repo, strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	server := httptest.NewServer(httpServer{app: a}.buildMux())

	defer server.Close()

	get := getJSON(t, server.URL+"/v1/settings")

	if valid, _ := get["valid"].(bool); !valid {
		t.Fatalf("settings response = %#v", get)
	}

	settings, _ := get["settings"].(map[string]any)

	if settings["repoRoot"] != repo {
		t.Fatalf("repoRoot = %#v", settings["repoRoot"])
	}

	body := bytes.NewBufferString(`{"settings":{"repoRoot":"` + filepath.Join(home, "missing") + `"}}`)
	resp, err := http.Post(server.URL+"/v1/settings/validate", "application/json", body)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	var validation map[string]any

	if err := json.NewDecoder(resp.Body).Decode(&validation); err != nil {
		t.Fatal(err)
	}

	if valid, _ := validation["valid"].(bool); valid {
		t.Fatalf("expected invalid settings, got %#v", validation)
	}
}

func TestHTTPHealthz(t *testing.T) {
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	server := httptest.NewServer(httpServer{app: a}.buildMux())

	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/healthz")

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func getJSON(t *testing.T, url string) map[string]any {
	t.Helper()

	resp, err := http.Get(url)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		t.Fatalf("GET %s = %d: %s", url, resp.StatusCode, body)
	}

	var body map[string]any

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	return body
}

func streamSSE(t *testing.T, url, body string) []sseEvent {
	t.Helper()

	resp, err := http.Post(url, "application/json", strings.NewReader(body))

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)

		t.Fatalf("POST %s = %d: %s", url, resp.StatusCode, raw)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var events []sseEvent
	current := sseEvent{}

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case line == "":
			if current.Event != "" || current.Data != "" {
				events = append(events, current)
				current = sseEvent{}
			}
		case strings.HasPrefix(line, "event: "):
			current.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			current.Data = strings.TrimPrefix(line, "data: ")
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	if current.Event != "" || current.Data != "" {
		events = append(events, current)
	}

	return events
}

func firstSSERunID(t *testing.T, events []sseEvent) string {
	t.Helper()

	for _, event := range events {
		if event.Event != "workflow" {
			continue
		}

		var payload storage.EventRecord

		if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
			continue
		}

		if payload.RunID != "" {
			return payload.RunID
		}
	}

	t.Fatalf("no run id in events %#v", events)

	return ""
}

func hasSSEEventType(events []sseEvent, eventType string) bool {
	for _, payload := range workflowSSEEvents(nil, events) {
		if payload.Type == eventType {
			return true
		}
	}

	return false
}

func workflowSSEEvents(t *testing.T, events []sseEvent) []storage.EventRecord {
	if t != nil {
		t.Helper()
	}

	var payloads []storage.EventRecord

	for _, event := range events {
		if event.Event != "workflow" {
			continue
		}

		var payload storage.EventRecord

		if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
			if t != nil {
				t.Fatalf("decode workflow event: %v", err)
			}

			continue
		}

		payloads = append(payloads, payload)
	}

	return payloads
}

func eventTypeIndex(events []storage.EventRecord, eventType string) int {
	for index, event := range events {
		if event.Type == eventType {
			return index
		}
	}

	return -1
}
