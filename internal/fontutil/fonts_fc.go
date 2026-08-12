//go:build linux || darwin

// Shared font enumeration helpers for Linux and macOS.
package fontutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// fontFamiliesFromFCList runs fc-list to enumerate font families.
// It returns nil when the command cannot be executed.
func fontFamiliesFromFCList() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "fc-list", "--format=%{family}\n").Output()
	if err != nil {
		return nil
	}

	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		for _, part := range strings.Split(line, ",") {
			if name := strings.TrimSpace(part); name != "" {
				names = append(names, name)
			}
		}
	}
	return uniqueSorted(names)
}

// scanFontDirs walks the given directories and collects font files
// (extensions .ttf/.otf/.ttc) whose base names, without extension, are
// used as font family names. A directory of "~/.fonts" is expanded to
// the current user's home directory; walk errors are ignored.
func scanFontDirs(dirs ...string) []string {
	var names []string
	for _, dir := range dirs {
		if dir == "~/.fonts" {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			dir = filepath.Join(home, ".fonts")
		}
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".ttf", ".otf", ".ttc":
				names = append(names, strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())))
			}
			return nil
		})
	}
	return uniqueSorted(names)
}

// uniqueSorted removes duplicate names and returns them sorted.
func uniqueSorted(names []string) []string {
	seen := make(map[string]bool, len(names))
	uniq := make([]string, 0, len(names))
	for _, name := range names {
		if !seen[name] {
			seen[name] = true
			uniq = append(uniq, name)
		}
	}
	sort.Strings(uniq)
	return uniq
}
