package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gocanto/dot-files/internal/domain"
	"github.com/gocanto/dot-files/internal/storage/db"
	_ "modernc.org/sqlite"
)

const DefaultTheme = "light"

//go:embed schema.sql
var schemaFS embed.FS

type Store struct {
	db      *sql.DB
	queries *db.Queries
	now     func() time.Time
}

type RunStart struct {
	ID                      string
	WorkflowID              string
	WorkflowName            string
	ConfirmationOptionID    string
	ConfirmationOptionLabel string
	Mode                    domain.RunMode
	Status                  domain.RunStatus
}

type RunSummary struct {
	ID                      string `json:"id"`
	WorkflowID              string `json:"workflowId"`
	WorkflowName            string `json:"workflowName"`
	ConfirmationOptionID    string `json:"confirmationOptionId"`
	ConfirmationOptionLabel string `json:"confirmationOptionLabel"`
	Mode                    string `json:"mode"`
	Status                  string `json:"status"`
	StartedAt               string `json:"startedAt"`
	CompletedAt             string `json:"completedAt,omitempty"`
	ErrorMessage            string `json:"errorMessage,omitempty"`
}

type EventRecord struct {
	ID        int64  `json:"id"`
	RunID     string `json:"runId"`
	Seq       int64  `json:"seq"`
	Type      string `json:"type"`
	PhaseID   string `json:"phaseId,omitempty"`
	PhaseName string `json:"phaseName,omitempty"`
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"createdAt"`
}

type RunLog struct {
	Run    RunSummary    `json:"run"`
	Events []EventRecord `json:"events"`
}

type UserPreferences struct {
	Theme     string `json:"theme"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type Recorder struct {
	store *Store
	runID string
	mu    sync.Mutex
	seq   int64
	also  func(domain.Event) error
}

const envDBPath = "DOT_FILES_WORKFLOW_DB_PATH"

func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite", path)

	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &Store{db: conn, queries: db.New(conn), now: time.Now}

	if err := store.Init(ctx); err != nil {
		_ = conn.Close()

		return nil, err
	}

	return store, nil
}

func DefaultPath(home string) string {
	if override := os.Getenv(envDBPath); override != "" {
		return override
	}

	return filepath.Join(home, "Library", "Application Support", "gus-mac", "workflows.sqlite3")
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Init(ctx context.Context) error {
	schema, err := schemaFS.ReadFile("schema.sql")

	if err != nil {
		return fmt.Errorf("read embedded sqlite schema: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, string(schema)); err != nil {
		return fmt.Errorf("initialize sqlite schema: %w", err)
	}

	return nil
}

func (s *Store) CreateRun(ctx context.Context, run RunStart) error {
	return s.queries.CreateRun(ctx, db.CreateRunParams{
		ID:                      run.ID,
		WorkflowID:              run.WorkflowID,
		WorkflowName:            run.WorkflowName,
		ConfirmationOptionID:    run.ConfirmationOptionID,
		ConfirmationOptionLabel: run.ConfirmationOptionLabel,
		Mode:                    string(run.Mode),
		Status:                  string(run.Status),
		StartedAt:               s.now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Store) CompleteRun(ctx context.Context, id string, status domain.RunStatus, message string) error {
	return s.queries.CompleteRun(ctx, db.CompleteRunParams{
		ID:           id,
		Status:       string(status),
		CompletedAt:  nullString(s.now().UTC().Format(time.RFC3339Nano)),
		ErrorMessage: nullString(message),
	})
}

func (s *Store) InsertEvent(ctx context.Context, event domain.Event) error {
	return s.queries.InsertEvent(ctx, db.InsertEventParams{
		RunID:     event.RunID,
		Seq:       event.Seq,
		EventType: event.Type,
		PhaseID:   nullString(event.PhaseID),
		PhaseName: nullString(event.PhaseName),
		Status:    nullString(event.Status),
		Message:   nullString(event.Message),
		CreatedAt: s.now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Store) ListRuns(ctx context.Context, limit int64) ([]RunSummary, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.queries.ListRuns(ctx, limit)

	if err != nil {
		return nil, err
	}

	runs := make([]RunSummary, 0, len(rows))

	for _, row := range rows {
		runs = append(runs, runSummary(row))
	}

	return runs, nil
}

func (s *Store) RunLog(ctx context.Context, runID string) (RunLog, error) {
	run, err := s.queries.GetRun(ctx, runID)

	if err != nil {
		return RunLog{}, err
	}

	rows, err := s.queries.ListRunEvents(ctx, runID)

	if err != nil {
		return RunLog{}, err
	}

	events := make([]EventRecord, 0, len(rows))

	for _, row := range rows {
		events = append(events, eventRecord(row))
	}

	return RunLog{Run: runSummary(run), Events: events}, nil
}

func (s *Store) GetUserPreferences(ctx context.Context) (UserPreferences, error) {
	row, err := s.queries.GetUserPreferences(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return UserPreferences{Theme: DefaultTheme}, nil
	}

	if err != nil {
		return UserPreferences{}, err
	}

	return UserPreferences{Theme: row.Theme, UpdatedAt: row.UpdatedAt}, nil
}

func (s *Store) SaveUserPreferences(ctx context.Context, prefs UserPreferences) (UserPreferences, error) {
	theme := prefs.Theme

	if theme == "" {
		theme = DefaultTheme
	}

	updatedAt := s.now().UTC().Format(time.RFC3339Nano)

	if err := s.queries.UpsertUserPreferences(ctx, db.UpsertUserPreferencesParams{
		Theme:     theme,
		UpdatedAt: updatedAt,
	}); err != nil {
		return UserPreferences{}, err
	}

	return UserPreferences{Theme: theme, UpdatedAt: updatedAt}, nil
}

func NewRecorder(store *Store, runID string, also func(domain.Event) error) *Recorder {
	return &Recorder{store: store, runID: runID, also: also}
}

func (r *Recorder) Emit(ctx context.Context, event domain.Event) error {
	r.mu.Lock()

	defer r.mu.Unlock()

	r.seq++
	event.RunID = r.runID
	event.Seq = r.seq

	if err := r.store.InsertEvent(ctx, event); err != nil {
		return err
	}

	if r.also != nil {
		return r.also(event)
	}

	return nil
}

func runSummary(row db.WorkflowRun) RunSummary {
	return RunSummary{
		ID:                      row.ID,
		WorkflowID:              row.WorkflowID,
		WorkflowName:            row.WorkflowName,
		ConfirmationOptionID:    row.ConfirmationOptionID,
		ConfirmationOptionLabel: row.ConfirmationOptionLabel,
		Mode:                    row.Mode,
		Status:                  row.Status,
		StartedAt:               row.StartedAt,
		CompletedAt:             fromNull(row.CompletedAt),
		ErrorMessage:            fromNull(row.ErrorMessage),
	}
}

func eventRecord(row db.WorkflowEvent) EventRecord {
	return EventRecord{
		ID:        row.ID,
		RunID:     row.RunID,
		Seq:       row.Seq,
		Type:      row.EventType,
		PhaseID:   fromNull(row.PhaseID),
		PhaseName: fromNull(row.PhaseName),
		Status:    fromNull(row.Status),
		Message:   fromNull(row.Message),
		CreatedAt: row.CreatedAt,
	}
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func fromNull(value sql.NullString) string {
	if !value.Valid {
		return ""
	}

	return value.String
}
