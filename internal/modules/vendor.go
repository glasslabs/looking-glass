package modules

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"
)

// Dependencies reads module dependencies from a vendored modules file.
func Dependencies(path string) ([]module.Version, error) {
	path = filepath.Join(path, "vendor", "modules.txt")
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}

	vendor, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var deps []module.Version
	for _, line := range strings.Split(strings.TrimSpace(string(vendor)), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 3 || parts[0] != "#" {
			continue
		}

		deps = append(deps, module.Version{Path: parts[1], Version: parts[2]})
	}

	return deps, nil
}
