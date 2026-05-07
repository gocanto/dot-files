package lineparse

import "strings"

func UniqueTrimmed(out []byte) []string {
	lines := strings.Split(string(out), "\n")
	items := make([]string, 0, len(lines))
	seen := map[string]struct{}{}

	for _, line := range lines {
		item := strings.TrimSpace(line)

		if item == "" {
			continue
		}

		if _, ok := seen[item]; ok {
			continue
		}

		seen[item] = struct{}{}
		items = append(items, item)
	}

	return items
}
