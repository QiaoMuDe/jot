// Font enumeration implementation for Linux: fc-list with a directory
// scan fallback.
package fontutil

// GetFonts enumerates installed font families on Linux.
// It prefers fc-list output and falls back to scanning common font
// directories when fc-list is unavailable.
func GetFonts() []string {
	if names := fontFamiliesFromFCList(); len(names) > 0 {
		return names
	}
	return scanFontDirs("/usr/share/fonts", "/usr/local/share/fonts", "~/.fonts")
}
