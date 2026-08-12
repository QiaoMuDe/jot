// Font enumeration implementation for macOS: fc-list with
// system_profiler and directory scan fallbacks.
package fontutil

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

// GetFonts enumerates installed font families on macOS.
// It tries fc-list first, then system_profiler, and finally scans
// common font directories.
func GetFonts() []string {
	if names := fontFamiliesFromFCList(); len(names) > 0 {
		return names
	}
	if names := fontFamiliesFromSystemProfiler(); len(names) > 0 {
		return names
	}
	return scanFontDirs("/System/Library/Fonts", "/Library/Fonts", "~/.fonts")
}

// fontFamiliesFromSystemProfiler runs system_profiler SPFontsDataType
// and extracts the "family" field of each font entry.
// It returns nil when the command fails or the output cannot be parsed.
func fontFamiliesFromSystemProfiler() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "system_profiler", "SPFontsDataType", "-json").Output()
	if err != nil {
		return nil
	}

	var data struct {
		SPFontsDataType []struct {
			Family string `json:"family"`
		} `json:"SPFontsDataType"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil
	}

	var names []string
	for _, entry := range data.SPFontsDataType {
		if name := strings.TrimSpace(entry.Family); name != "" {
			names = append(names, name)
		}
	}
	return uniqueSorted(names)
}
