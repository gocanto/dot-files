package apps

import (
	"sort"
	"strings"
)

type MASInstall = masInstall

func parseBrewCasks(out []byte) []string {
	lines := strings.Split(string(out), "\n")
	casks := make([]string, 0, len(lines))
	seen := map[string]bool{}

	for _, line := range lines {
		name := strings.TrimSpace(line)

		if name == "" || seen[name] {
			continue
		}

		seen[name] = true
		casks = append(casks, name)
	}

	sort.Slice(casks, func(i, j int) bool {
		return strings.ToLower(casks[i]) < strings.ToLower(casks[j])
	})

	return casks
}

func ParseMASList(out []byte) []MASInstall {
	return parseMASList(out)
}

func parseMASList(out []byte) []masInstall {
	lines := strings.Split(string(out), "\n")
	apps := make([]masInstall, 0, len(lines))
	seen := map[string]bool{}

	for _, line := range lines {
		fields := strings.Fields(line)

		if len(fields) < 2 {
			continue
		}

		id := fields[0]
		rest := strings.TrimSpace(strings.TrimPrefix(line, id))
		version := strings.LastIndex(rest, " (")

		if version >= 0 {
			rest = strings.TrimSpace(rest[:version])
		}

		if id == "" || rest == "" || seen[id] {
			continue
		}

		seen[id] = true
		apps = append(apps, masInstall{ID: id, Name: rest})
	}

	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})

	return apps
}
