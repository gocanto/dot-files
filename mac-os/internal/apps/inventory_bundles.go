package apps

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type plist struct {
	Dict plistDict `xml:"dict"`
}

type plistDict struct {
	Items []plistItem `xml:",any"`
}

type plistItem struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func scanAppBundles(roots []string) ([]installedBundle, error) {
	var bundles []installedBundle
	seen := map[string]bool{}

	for _, root := range roots {
		info, err := os.Stat(root)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, err
		}

		if !info.IsDir() {
			continue
		}

		systemRoot := strings.HasPrefix(filepath.Clean(root), "/System/")

		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".app" {
				return nil
			}

			metadata, err := readBundleMetadata(path)

			if err != nil {
				name := strings.TrimSuffix(filepath.Base(path), ".app")
				metadata = installedBundle{Name: name}
			}

			metadata.Path = path
			metadata.System = systemRoot || strings.HasPrefix(filepath.Clean(path), "/System/")

			key := metadata.BundleID

			if key == "" {
				key = normalize(metadata.Name)
			}

			if key == "" || seen[key] {
				return filepath.SkipDir
			}

			seen[key] = true
			bundles = append(bundles, metadata)

			return filepath.SkipDir
		})

		if err != nil {
			return nil, err
		}
	}

	sort.Slice(bundles, func(i, j int) bool {
		return strings.ToLower(bundles[i].Name) < strings.ToLower(bundles[j].Name)
	})

	return bundles, nil
}

func readBundleMetadata(appPath string) (installedBundle, error) {
	infoPath := filepath.Join(appPath, "Contents", "Info.plist")
	data, err := os.ReadFile(infoPath)

	if err != nil {
		return installedBundle{}, err
	}

	values := parsePlistStrings(data)

	if values["CFBundleIdentifier"] == "" {
		if converted, err := exec.Command("plutil", "-convert", "xml1", "-o", "-", infoPath).Output(); err == nil {
			values = parsePlistStrings(converted)
		}
	}

	name := firstNonEmpty(values["CFBundleDisplayName"], values["CFBundleName"], strings.TrimSuffix(filepath.Base(appPath), ".app"))

	return installedBundle{
		Name:     name,
		BundleID: values["CFBundleIdentifier"],
	}, nil
}

func parsePlistStrings(data []byte) map[string]string {
	values := map[string]string{}

	if !bytes.Contains(data, []byte("<plist")) {
		return values
	}

	var p plist

	if err := xml.Unmarshal(data, &p); err != nil {
		return values
	}

	var key string

	for _, item := range p.Dict.Items {
		switch item.XMLName.Local {
		case "key":
			key = strings.TrimSpace(item.Value)
		case "string":
			if key != "" {
				values[key] = strings.TrimSpace(item.Value)
			}

			key = ""
		default:
			key = ""
		}
	}

	return values
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
