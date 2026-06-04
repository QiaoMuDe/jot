// Package fontutil provides functions to enumerate system fonts on Windows
// using the GDI EnumFontFamiliesW API directly via syscall.
package fontutil

import (
	"sort"
	"syscall"
)

var (
	modGdi32  = syscall.NewLazyDLL("gdi32.dll")
	modUser32 = syscall.NewLazyDLL("user32.dll")

	procEnumFonts = modGdi32.NewProc("EnumFontFamiliesW")
	procGetDC     = modUser32.NewProc("GetDC")
	procReleaseDC = modUser32.NewProc("ReleaseDC")
)

// LOGFONTW corresponds to the Windows LOGFONTW structure.
type LOGFONTW struct {
	LfHeight         int32
	LfWidth          int32
	LfEscapement     int32
	LfOrientation    int32
	LfWeight         int32
	LfItalic         uint8
	LfUnderline      uint8
	LfStrikeOut      uint8
	LfCharSet        uint8
	LfOutPrecision   uint8
	LfClipPrecision  uint8
	LfQuality        uint8
	LfPitchAndFamily uint8
	LfFaceName       [32]uint16
}

// NEWTEXTMETRICEXW corresponds to the Windows NEWTEXTMETRICEXW structure.
type NEWTEXTMETRICEXW struct {
	TmHeight           int32
	TmAscent           int32
	TmDescent          int32
	TmInternalLeading  int32
	TmExternalLeading  int32
	TmAveCharWidth     int32
	TmMaxCharWidth     int32
	TmWeight           int32
	TmItalic           uint8
	TmUnderlined       uint8
	TmStruckOut        uint8
	TmFirstChar        uint8
	TmLastChar         uint8
	TmDefaultChar      uint8
	TmBreakChar        uint8
	TmPitchAndFamily   uint8
	TmCharSet          uint8
	TmOverhang         int32
	TmDigitizedAspectX int32
	TmDigitizedAspectY int32
	TmFaceName         [64]uint16
	FontIndex          uint32
	FontType           int32
}

// fonts and fontMap are used as temporary storage during enumeration.
// They are reset on each call to GetFonts.
var (
	fonts   []string
	fontMap map[string]bool
)

// GetFonts enumerates all installed font families on the system
// using the Windows GDI EnumFontFamiliesW API.
// Returns a sorted slice of unique font family names.
func GetFonts() []string {
	// Reset state for each call
	fonts = nil
	fontMap = make(map[string]bool)

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil
	}
	defer procReleaseDC.Call(0, hdc) //nolint:errcheck

	var lf LOGFONTW
	lf.LfCharSet = 0xFF // DEFAULT_CHARSET

	callback := syscall.NewCallback(enumFontCallback)
	procEnumFonts.Call(hdc, 0, callback, 0) //nolint:errcheck

	sort.Strings(fonts)
	return fonts
}

// enumFontCallback is the callback passed to EnumFontFamiliesW.
// It collects unique font family names into the package-level slices.
func enumFontCallback(lplf *LOGFONTW, lpntm *NEWTEXTMETRICEXW, fontType uint32, lParam uintptr) uintptr {
	// Skip fonts with no type (continuation)
	if fontType == 0 {
		return 1
	}

	name := syscall.UTF16ToString(lplf.LfFaceName[:])

	if fontMap[name] {
		return 1
	}
	fontMap[name] = true
	fonts = append(fonts, name)

	return 1 // continue enumeration
}
