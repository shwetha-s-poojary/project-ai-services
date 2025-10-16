package helpers

import (
	"os"
	"strings"
)

func GetTemplateNames(dir string) ([]string, error) {
	var names []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".yaml.tmpl") {
			// Strip the suffix
			base := strings.TrimSuffix(name, ".yaml.tmpl")
			names = append(names, base)
		}
	}

	return names, nil
}
