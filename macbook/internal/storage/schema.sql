CREATE TABLE IF NOT EXISTS workflow_runs (
  id TEXT PRIMARY KEY,
  workflow_id TEXT NOT NULL,
  workflow_name TEXT NOT NULL,
  confirmation_option_id TEXT NOT NULL,
  confirmation_option_label TEXT NOT NULL,
  mode TEXT NOT NULL,
  status TEXT NOT NULL,
  started_at TEXT NOT NULL,
  completed_at TEXT,
  error_message TEXT
);

CREATE TABLE IF NOT EXISTS workflow_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  phase_id TEXT,
  phase_name TEXT,
  status TEXT,
  message TEXT,
  created_at TEXT NOT NULL,
  UNIQUE(run_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_started_at ON workflow_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_workflow_events_run_id_seq ON workflow_events(run_id, seq);
