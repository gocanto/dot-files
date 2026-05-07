package setting

type RuntimeSettings struct {
	RepoRoot          string `json:"repoRoot"`
	AppsConfigPath    string `json:"appsConfigPath"`
	SecretsConfigPath string `json:"secretsConfigPath"`
	GeneratedAppsPath string `json:"generatedAppsPath"`
	ArchiveRoot       string `json:"archiveRoot"`
	WorkflowDBPath    string `json:"workflowDbPath"`
	OPVault           string `json:"opVault"`
	OPItem            string `json:"opItem"`
}

type Check struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Path    string `json:"path"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Validation struct {
	Settings RuntimeSettings `json:"settings"`
	Checks   []Check         `json:"checks"`
	Valid    bool            `json:"valid"`
}

const (
	CheckOK    = "ok"
	CheckError = "error"
)
