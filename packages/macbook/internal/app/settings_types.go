package app

type runtimeSettings struct {
	RepoRoot          string `json:"repoRoot"`
	AppsConfigPath    string `json:"appsConfigPath"`
	SecretsConfigPath string `json:"secretsConfigPath"`
	GeneratedAppsPath string `json:"generatedAppsPath"`
	ArchiveRoot       string `json:"archiveRoot"`
	WorkflowDBPath    string `json:"workflowDbPath"`
	OPVault           string `json:"opVault"`
	OPItem            string `json:"opItem"`
}

type settingsCheck struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Path    string `json:"path"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type settingsValidation struct {
	Settings runtimeSettings `json:"settings"`
	Checks   []settingsCheck `json:"checks"`
	Valid    bool            `json:"valid"`
}

const (
	checkOK    = "ok"
	checkError = "error"
)
