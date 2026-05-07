package services

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gocanto/dot-files/internal/safefs"
)

type templateFileSummary struct {
	Path       string `json:"path"`
	Relative   string `json:"relative"`
	Kind       string `json:"kind"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt,omitempty"`
	Exists     bool   `json:"exists"`
}

type templateFileContent struct {
	File    templateFileSummary `json:"file"`
	Content string              `json:"content"`
}

const maxTemplateFileBytes = 512 * 1024

func (s Service) ListTemplateFiles() ([]templateFileSummary, error) {
	allowed, err := s.templateFileAllowlist()

	if err != nil {
		return nil, err
	}

	files := make([]templateFileSummary, 0, len(allowed))

	for _, file := range allowed {
		summary, err := s.templateFileSummary(file)

		if err != nil {
			return nil, err
		}

		files = append(files, summary)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Relative < files[j].Relative
	})

	return files, nil
}

func (s Service) ReadTemplateFile(path string) (templateFileContent, error) {
	resolved, err := s.resolveTemplateFile(path)

	if err != nil {
		return templateFileContent{}, err
	}

	info, err := os.Stat(resolved)

	if err != nil {
		return templateFileContent{}, err
	}

	if err := validateTemplateFileInfo(resolved, info); err != nil {
		return templateFileContent{}, err
	}

	data, err := os.ReadFile(resolved)

	if err != nil {
		return templateFileContent{}, err
	}

	if err := validateTemplateFileBytes(data); err != nil {
		return templateFileContent{}, err
	}

	summary, err := s.templateFileSummary(resolved)

	if err != nil {
		return templateFileContent{}, err
	}

	return templateFileContent{File: summary, Content: string(data)}, nil
}

func (s Service) SaveTemplateFile(path, content string) (templateFileContent, error) {
	resolved, err := s.resolveTemplateFile(path)

	if err != nil {
		return templateFileContent{}, err
	}

	if len([]byte(content)) > maxTemplateFileBytes {
		return templateFileContent{}, fmt.Errorf("template file is too large: max %d bytes", maxTemplateFileBytes)
	}

	if !utf8.ValidString(content) || bytes.ContainsRune([]byte(content), 0) {
		return templateFileContent{}, errors.New("template file content must be UTF-8 text")
	}

	if info, err := os.Stat(resolved); err == nil {
		if err := validateTemplateFileInfo(resolved, info); err != nil {
			return templateFileContent{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return templateFileContent{}, err
	}

	if err := safefs.WriteFile(resolved, []byte(content), 0o600); err != nil {
		return templateFileContent{}, err
	}

	return s.ReadTemplateFile(resolved)
}

func (s Service) resolveTemplateFile(path string) (string, error) {
	path = strings.TrimSpace(path)

	if path == "" {
		return "", errors.New("path is required")
	}

	candidate := path

	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(s.Repo, candidate)
	}

	candidate = filepath.Clean(candidate)
	allowed, err := s.templateFileAllowlist()

	if err != nil {
		return "", err
	}

	for _, allowedPath := range allowed {
		if candidate == allowedPath {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("path is not an editable template file: %s", path)
}

func (s Service) templateFileAllowlist() ([]string, error) {
	paths := []string{
		filepath.Clean(s.Settings.AppsConfigPath),
		filepath.Clean(s.Settings.SecretsConfigPath),
		filepath.Clean(s.Settings.GeneratedAppsPath),
	}

	stowDir := filepath.Join(s.Repo, "stow")

	if err := filepath.WalkDir(stowDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == stowDir {
			return nil
		}

		if entry.IsDir() {
			if safefs.ShouldSkipSensitive(path) {
				return filepath.SkipDir
			}

			return nil
		}

		if entry.Type()&fs.ModeSymlink != 0 || !entry.Type().IsRegular() || safefs.ShouldSkipSensitive(path) {
			return nil
		}

		paths = append(paths, filepath.Clean(path))

		return nil
	}); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	unique := make([]string, 0, len(paths))

	for _, path := range paths {
		if seen[path] {
			continue
		}

		seen[path] = true
		unique = append(unique, path)
	}

	return unique, nil
}

func (s Service) templateFileSummary(path string) (templateFileSummary, error) {
	relative, err := filepath.Rel(s.Repo, path)

	if err != nil || strings.HasPrefix(relative, "..") {
		relative = path
	}

	summary := templateFileSummary{
		Path:     path,
		Relative: filepath.ToSlash(relative),
		Kind:     s.templateFileKind(path),
		Exists:   false,
	}

	info, err := os.Stat(path)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return summary, nil
		}

		return summary, err
	}

	summary.Exists = true
	summary.Size = info.Size()
	summary.ModifiedAt = info.ModTime().UTC().Format(time.RFC3339)

	return summary, nil
}

func (s Service) templateFileKind(path string) string {
	switch path {
	case filepath.Clean(s.Settings.AppsConfigPath), filepath.Clean(s.Settings.GeneratedAppsPath):
		return "apps"
	case filepath.Clean(s.Settings.SecretsConfigPath):
		return "secrets"
	default:
		return "stow"
	}
}

func validateTemplateFileInfo(path string, info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("template file is not a regular file: %s", path)
	}

	if info.Size() > maxTemplateFileBytes {
		return fmt.Errorf("template file is too large: max %d bytes", maxTemplateFileBytes)
	}

	return nil
}

func validateTemplateFileBytes(data []byte) error {
	if len(data) > maxTemplateFileBytes {
		return fmt.Errorf("template file is too large: max %d bytes", maxTemplateFileBytes)
	}

	if !utf8.Valid(data) || bytes.ContainsRune(data, 0) {
		return errors.New("template file must be UTF-8 text")
	}

	return nil
}
