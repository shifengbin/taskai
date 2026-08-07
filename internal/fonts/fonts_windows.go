//go:build windows

package fonts

import (
	"syscall"
	"unsafe"
)

const (
	defaultCharSet = 1
	tmpfFixedPitch = 0x01
	ffMask         = 0xf0
	ffModern       = 0x30
	lfFaceSize     = 32
)

var (
	gdi32                   = syscall.NewLazyDLL("gdi32.dll")
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumFontFamiliesExW = gdi32.NewProc("EnumFontFamiliesExW")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	fontFamilyEnumerationCB = syscall.NewCallback(enumFontFamily)
)

type logfontW struct {
	Height         int32
	Width          int32
	Escapement     int32
	Orientation    int32
	Weight         int32
	Italic         byte
	Underline      byte
	StrikeOut      byte
	CharSet        byte
	OutPrecision   byte
	ClipPrecision  byte
	Quality        byte
	PitchAndFamily byte
	FaceName       [lfFaceSize]uint16
}

type textmetricW struct {
	Height           int32
	Ascent           int32
	Descent          int32
	InternalLeading  int32
	ExternalLeading  int32
	AveCharWidth     int32
	MaxCharWidth     int32
	Weight           int32
	Overhang         int32
	DigitizedAspectX int32
	DigitizedAspectY int32
	FirstChar        uint16
	LastChar         uint16
	DefaultChar      uint16
	BreakChar        uint16
	Italic           byte
	Underlined       byte
	StruckOut        byte
	PitchAndFamily   byte
	CharSet          byte
}

func systemCandidates() []Candidate {
	dc, _, _ := procGetDC.Call(0)
	if dc == 0 {
		return nil
	}
	defer procReleaseDC.Call(0, dc)

	result := make([]Candidate, 0)
	font := logfontW{CharSet: defaultCharSet}
	_, _, _ = procEnumFontFamiliesExW.Call(
		dc,
		uintptr(unsafe.Pointer(&font)),
		fontFamilyEnumerationCB,
		uintptr(unsafe.Pointer(&result)),
		0,
	)
	return result
}

func enumFontFamily(logfontPointer, textmetricPointer, _ uintptr, parameter uintptr) uintptr {
	metric := (*textmetricW)(unsafe.Pointer(textmetricPointer))
	if metric.PitchAndFamily&tmpfFixedPitch != 0 && metric.PitchAndFamily&ffMask != ffModern {
		return 1
	}
	font := (*logfontW)(unsafe.Pointer(logfontPointer))
	family := syscall.UTF16ToString(font.FaceName[:])
	if family == "" {
		return 1
	}
	result := (*[]Candidate)(unsafe.Pointer(parameter))
	*result = append(*result, Candidate{Family: family, Spacing: SpacingMono})
	return 1
}
