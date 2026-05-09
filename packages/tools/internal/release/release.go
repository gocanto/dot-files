package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocanto/dot-files/tools/internal/runner"
)

type Tool struct {
	RootDir string
	Config  Config
	Runner  runner.CommandRunner
	Stdout  io.Writer
}

type uiManifest struct {
	Version string `json:"version"`
}

type artifacts struct {
	DMG string
	ZIP string
}

type workflow struct {
	rootDir string
	config  Config
	runner  runner.CommandRunner
	stdout  io.Writer
}

func (t Tool) Run(ctx context.Context) error {
	commandRunner := t.Runner

	if commandRunner == nil {
		commandRunner = runner.ExecRunner{}
	}

	stdout := t.Stdout

	if stdout == nil {
		stdout = os.Stdout
	}

	workflow := workflow{
		rootDir: t.RootDir,
		config:  t.Config,
		runner:  commandRunner,
		stdout:  stdout,
	}

	return workflow.run(ctx)
}

func (w workflow) run(ctx context.Context) error {
	if err := w.validateConfig(); err != nil {
		return err
	}

	if err := w.validateRequiredCommands(); err != nil {
		return err
	}

	repo, err := w.releaseRepo(ctx)

	if err != nil {
		return err
	}

	version, err := readUIVersion(filepath.Join(w.rootDir, "packages", "ui", "package.json"))

	if err != nil {
		return err
	}

	tag := w.config.Tag

	if tag == "" {
		tag = "v" + version
	}

	head, err := w.output(ctx, "git", "rev-parse", "HEAD")

	if err != nil {
		return err
	}

	defaultBranch, currentBranch, err := w.branches(ctx)

	if err != nil {
		return err
	}

	if currentBranch != defaultBranch {
		return fmt.Errorf("releases must be cut from %s, currently on %s", defaultBranch, currentBranch)
	}

	if err := w.validateCleanTree(ctx); err != nil {
		return err
	}

	fmt.Fprintln(w.stdout, "Fetching origin and tags...")

	if err := w.runCommand(ctx, "git", "fetch", "origin", defaultBranch, "--tags"); err != nil {
		return err
	}

	if err := w.validateRemoteState(ctx, defaultBranch); err != nil {
		return err
	}

	if err := w.validateTagAvailable(ctx, tag); err != nil {
		return err
	}

	fmt.Fprintln(w.stdout, "Running tests and builds...")

	if err := w.runBuilds(ctx); err != nil {
		return err
	}

	releaseDir := filepath.Join(w.rootDir, "packages", "ui", "release")
	foundArtifacts, err := findArtifacts(releaseDir)

	if err != nil {
		return err
	}

	foundArtifacts, err = versionArtifactsForTag(foundArtifacts, version, tag)

	if err != nil {
		return err
	}

	checksumsFile, err := w.writeChecksums(ctx, releaseDir, foundArtifacts)

	if err != nil {
		return err
	}

	releaseNotes, cleanup, err := writeReleaseNotes(w.config.NotesFile)

	if err != nil {
		return err
	}

	defer cleanup()

	fmt.Fprintf(w.stdout, "Creating published release %s on %s...\n", tag, repo)

	if err := w.runCommand(ctx, "gh", "release", "create", tag, foundArtifacts.DMG, foundArtifacts.ZIP, checksumsFile, "--repo", repo, "--target", head, "--title", fmt.Sprintf("macOS Manager %s", tag), "--notes-file", releaseNotes); err != nil {
		return err
	}

	exists, err := w.localTagExists(ctx, tag)

	if err != nil {
		return err
	}

	if !exists {
		if err := w.runCommand(ctx, "git", "tag", tag, head); err != nil {
			return err
		}
	}

	if err := w.runCommand(ctx, "git", "push", "origin", tag); err != nil {
		return err
	}

	releaseURL, err := w.output(ctx, "gh", "release", "view", tag, "--repo", repo, "--json", "url", "--jq", ".url")

	if err != nil {
		return err
	}

	fmt.Fprintf(w.stdout, "Release published: %s\n", releaseURL)

	return nil
}

func (w workflow) validateConfig() error {
	if w.config.NotesFile == "" {
		return errors.New("--notes-file <path> is required")
	}

	info, err := os.Stat(w.config.NotesFile)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("notes file is missing or empty: %s", w.config.NotesFile)
		}

		return fmt.Errorf("inspect notes file: %w", err)
	}

	if info.IsDir() || info.Size() == 0 {
		return fmt.Errorf("notes file is missing or empty: %s", w.config.NotesFile)
	}

	return nil
}

func (w workflow) validateRequiredCommands() error {
	for _, command := range []string{"git", "gh", "pnpm", "node", "shasum"} {
		if err := w.runner.LookPath(command); err != nil {
			return err
		}
	}

	return nil
}

func (w workflow) releaseRepo(ctx context.Context) (string, error) {
	if w.config.Repo != "" {
		return w.config.Repo, nil
	}

	return w.output(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
}

func (w workflow) branches(ctx context.Context) (string, string, error) {
	currentBranch, err := w.output(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")

	if err != nil {
		return "", "", err
	}

	defaultBranch, err := w.output(ctx, "git", "symbolic-ref", "--short", "refs/remotes/origin/HEAD")

	if err != nil {
		defaultBranch = "main"
	} else {
		defaultBranch = strings.TrimPrefix(defaultBranch, "origin/")

		if defaultBranch == "" {
			defaultBranch = "main"
		}
	}

	return defaultBranch, currentBranch, nil
}

func (w workflow) validateCleanTree(ctx context.Context) error {
	err := w.runCommand(ctx, "git", "diff-index", "--quiet", "HEAD", "--")

	if err == nil {
		return nil
	}

	var exitErr *runner.ExitError

	if !errors.As(err, &exitErr) {
		return err
	}

	status, statusErr := w.output(ctx, "git", "status", "--short")

	if statusErr != nil {
		return errors.New("working tree has uncommitted changes")
	}

	if status != "" {
		return fmt.Errorf("working tree has uncommitted changes\n%s", status)
	}

	return errors.New("working tree has uncommitted changes")
}

func (w workflow) validateRemoteState(ctx context.Context, defaultBranch string) error {
	localRev, err := w.output(ctx, "git", "rev-parse", "HEAD")

	if err != nil {
		return err
	}

	remoteRev, err := w.output(ctx, "git", "rev-parse", "origin/"+defaultBranch)

	if err != nil {
		return err
	}

	baseRev, err := w.output(ctx, "git", "merge-base", "HEAD", "origin/"+defaultBranch)

	if err != nil {
		return err
	}

	if localRev != remoteRev && baseRev != remoteRev {
		return fmt.Errorf("local %s is not a fast-forward of origin/%s\n  local:  %s\n  origin: %s", defaultBranch, defaultBranch, localRev, remoteRev)
	}

	return nil
}

func (w workflow) validateTagAvailable(ctx context.Context, tag string) error {
	exists, err := w.localTagExists(ctx, tag)

	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("tag %s already exists locally", tag)
	}

	_, err = w.output(ctx, "git", "ls-remote", "--tags", "--exit-code", "origin", "refs/tags/"+tag)

	if err == nil {
		return fmt.Errorf("tag %s already exists on origin", tag)
	}

	var exitErr *runner.ExitError

	if errors.As(err, &exitErr) && exitErr.ExitCode == 2 {
		return nil
	}

	return err
}

func (w workflow) localTagExists(ctx context.Context, tag string) (bool, error) {
	_, err := w.output(ctx, "git", "rev-parse", "--verify", "--quiet", "refs/tags/"+tag)

	if err == nil {
		return true, nil
	}

	var exitErr *runner.ExitError

	if errors.As(err, &exitErr) {
		return false, nil
	}

	return false, err
}

func (w workflow) runBuilds(ctx context.Context) error {
	macbookDir := filepath.Join(w.rootDir, "packages", "macbook")
	uiDir := filepath.Join(w.rootDir, "packages", "ui")

	commands := []runner.CommandSpec{
		{Cwd: w.rootDir, Name: "pnpm", Args: []string{"-C", w.rootDir, "test"}},
		{Cwd: w.rootDir, Name: "pnpm", Args: []string{"-C", w.rootDir, "build"}},
		{Cwd: w.rootDir, Name: "pnpm", Args: []string{"--dir", macbookDir, "run", "build"}},
		{Cwd: w.rootDir, Name: "pnpm", Args: []string{"--dir", uiDir, "run", "dist:mac:unsigned"}},
	}

	for _, command := range commands {
		if err := w.runner.Run(ctx, command); err != nil {
			return err
		}
	}

	return nil
}

func (w workflow) writeChecksums(ctx context.Context, releaseDir string, found artifacts) (string, error) {
	checksumsFile := filepath.Join(releaseDir, "SHASUMS256.txt")
	file, err := os.Create(checksumsFile)

	if err != nil {
		return "", fmt.Errorf("create checksums file: %w", err)
	}

	defer file.Close()

	err = w.runner.Run(ctx, runner.CommandSpec{
		Cwd:    releaseDir,
		Name:   "shasum",
		Args:   []string{"-a", "256", filepath.Base(found.DMG), filepath.Base(found.ZIP)},
		Stdout: file,
	})

	if err != nil {
		return "", err
	}

	return checksumsFile, nil
}

func (w workflow) runCommand(ctx context.Context, name string, args ...string) error {
	return w.runner.Run(ctx, runner.CommandSpec{Cwd: w.rootDir, Name: name, Args: args})
}

func (w workflow) output(ctx context.Context, name string, args ...string) (string, error) {
	return w.runner.Output(ctx, runner.CommandSpec{Cwd: w.rootDir, Name: name, Args: args})
}

func readUIVersion(path string) (string, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return "", fmt.Errorf("read UI package manifest: %w", err)
	}

	var manifest uiManifest

	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("parse UI package manifest: %w", err)
	}

	if manifest.Version == "" {
		return "", errors.New("UI package manifest is missing version")
	}

	return manifest.Version, nil
}

func findArtifacts(releaseDir string) (artifacts, error) {
	entries, err := os.ReadDir(releaseDir)

	if err != nil {
		return artifacts{}, fmt.Errorf("read release directory %s: %w", releaseDir, err)
	}

	var found artifacts

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(releaseDir, entry.Name())

		switch {
		case found.DMG == "" && strings.EqualFold(filepath.Ext(entry.Name()), ".dmg"):
			found.DMG = path
		case found.ZIP == "" && strings.EqualFold(filepath.Ext(entry.Name()), ".zip"):
			found.ZIP = path
		}
	}

	if found.DMG == "" || found.ZIP == "" {
		return artifacts{}, fmt.Errorf("expected DMG and ZIP artifacts in %s", releaseDir)
	}

	return found, nil
}

func versionArtifactsForTag(found artifacts, version string, tag string) (artifacts, error) {
	dmg, err := versionArtifactForTag(found.DMG, version, tag)

	if err != nil {
		return artifacts{}, err
	}

	zip, err := versionArtifactForTag(found.ZIP, version, tag)

	if err != nil {
		return artifacts{}, err
	}

	return artifacts{DMG: dmg, ZIP: zip}, nil
}

func versionArtifactForTag(path string, version string, tag string) (string, error) {
	dir := filepath.Dir(path)
	name := filepath.Base(path)
	nextName := strings.Replace(name, version, tag, 1)

	if nextName == name {
		return "", fmt.Errorf("artifact %s does not include package version %s", path, version)
	}

	nextPath := filepath.Join(dir, nextName)

	if nextPath == path {
		return path, nil
	}

	if _, err := os.Stat(nextPath); err == nil {
		return "", fmt.Errorf("tagged artifact already exists: %s", nextPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect tagged artifact: %w", err)
	}

	if err := os.Rename(path, nextPath); err != nil {
		return "", fmt.Errorf("rename artifact for release tag: %w", err)
	}

	return nextPath, nil
}

func writeReleaseNotes(notesFile string) (string, func(), error) {
	tempFile, err := os.CreateTemp("", "release-notes-*.md")

	if err != nil {
		return "", nil, fmt.Errorf("create release notes file: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(tempFile.Name())
	}

	prefix := "> These macOS artifacts are unsigned while Developer ID approval is pending.\n" +
		"> On first launch, use right-click > Open, or remove quarantine manually:\n" +
		"> `xattr -dr com.apple.quarantine \"/Applications/macOS Manager.app\"`\n\n"

	if _, err := tempFile.WriteString(prefix); err != nil {
		tempFile.Close()
		cleanup()

		return "", nil, fmt.Errorf("write release notes preface: %w", err)
	}

	notes, err := os.Open(notesFile)

	if err != nil {
		tempFile.Close()
		cleanup()

		return "", nil, fmt.Errorf("open release notes: %w", err)
	}

	defer notes.Close()

	if _, err := io.Copy(tempFile, notes); err != nil {
		tempFile.Close()
		cleanup()

		return "", nil, fmt.Errorf("append release notes: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		cleanup()

		return "", nil, fmt.Errorf("close release notes file: %w", err)
	}

	return tempFile.Name(), cleanup, nil
}
