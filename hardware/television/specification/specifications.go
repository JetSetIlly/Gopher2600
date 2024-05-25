// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

// Package specification contains the definitions, including colour, of the PAL
// and NTSC television protocols supported by the emulation.
package specification

import (
	"image/color"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// SpecList is the list of specifications that the television may adopt.
var SpecList = []string{"NTSC", "PAL", "PAL-M", "SECAM"}

// ReqSpecList is the list of specifications that can be requested. This is
// different to the actual spec list in that it includes "PAL60" and "AUTO" as
// an option
var ReqSpecList = slices.Concat([]string{"AUTO"}, SpecList, []string{"PAL60"})

// NormaliseReqSpecID converts the ID string such that it is one of the values
// in ReqSpecList. If the ID is not in ReqSpecList then false is returned.
//
// For completeness, the empty string is converted to "AUTO". "PALM" is
// converted to "PAL-M"; and "PAL-60" converted to "PAL60".
func NormaliseReqSpecID(id string) (string, bool) {
	id = strings.ToUpper(id)
	switch id {
	case "":
		id = "AUTO"
	case "PALM":
		id = "PAL-M"
	case "PAL-60":
		id = "PAL60"
	}
	return id, slices.Index(ReqSpecList, id) != -1
}

// SearchReqSpec looks for a valid sub-string in s, that indicates a required TV
// specification. The returned value is one listed in ReqSpecList or the empty
// string to indicate that nothing was found in the supplied string.
//
// This function is intended to be used for searching filenames or descriptions.
// It probably shouldn't be used as a general conversion tool.
func SearchReqSpec(s string) string {
	var id string

	// we don't want to include the path in the search because this may cause
	// false positives. for example, in ROM Hunter's archive there are
	// directories called "PAL VERSIONS OF NTSC ORIGINALS" and "NTSC VERSIONS OF
	// PAL ORIGINALS"
	//
	// http://www.atarimania.com/rom_collection_archive_atari_2600_roms.html
	s = filepath.Base(s)

	// look for spec substring in the supplied string
	s = strings.ToUpper(s)
	for _, spec := range ReqSpecList {
		if strings.Contains(s, spec) {
			switch spec {
			case "AUTO":
				// ignore appearance of auto in the string
			case "PAL-60":
				id = "PAL60"
			case "PAL60":
				id = "PAL60"
			case "PAL-M":
				id = "PAL-M"
			case "PALM":
				id = "PAL-M"
			case "NTSC":
				id = "NTSC"
			case "PAL":
				id = "PAL"
			case "SECAM":
				id = "SECAM"
			}
		}
	}

	return id
}

// Spec is used to define the two television specifications.
type Spec struct {
	ID     string
	Colors []color.RGBA

	// horizontal scan rate is used to calculate the refresh rate figure
	HorizontalScanRate float32

	// the nominal refresh rate for the specification. this refresh rate will
	// be produced if the actual number of scanlines per frame is the same as
	// OptimalTotal defined below.
	RefreshRate float32

	// the number of scanlines the 2600 Programmer's guide recommends for the
	// top/bottom parts of the screen:
	//
	// "A typical frame will consists of 3 vertical sync (VSYNC) lines*, 37 vertical
	// blank (VBLANK) lines, 192 TV picture lines, and 30 overscan lines. Atari’s
	// research has shown that this pattern will work on all types of TV sets."
	//
	// the quoted figures above are in reference to the NTSC protocol
	ScanlinesVSync    int
	ScanlinesVBlank   int
	ScanlinesVisible  int
	ScanlinesOverscan int

	// the optimal number of total scanlines for the entire frame. is the sum of
	// the four regions defined above.
	//
	// if the actual TV frame has a different number than this then the refresh
	// rate will not be optimal.
	ScanlinesTotal int

	// the scanline at which the VBLANK should be turned off (Top) and
	// turned back on again (Bottom). the period between the top and bottom
	// scanline is the visible portion of the screen.
	//
	// in practice, the VCS can turn VBLANK on and off at any time; what the
	// two values below represent what "Atari's research" (according to page 1
	// of the "Stella Programmer's Guide") has shown to be safe. by definition
	// this means that:
	//
	//	Top = VSync + Vblank
	//
	//	Bottom = Top + Visible
	//
	// or
	//
	//	Bottom = Total - Overscan
	AtariSafeVisibleTop    int
	AtariSafeVisibleBottom int

	// resizing of the TV is problematic because we can't rely on the VBLANK to
	// tell us when the pixels are meant to be in view. The NewSafeVisibleTop an
	// SafeBottom are the min/max values that the resizer should allow.
	//
	// think of these as the "modern" safe values as compared to the Atari
	// defined safe values.
	NewSafeVisibleTop    int
	NewSafeVisibleBottom int
}

// GetColor translates a signals to the color type.
func (spec *Spec) GetColor(col signal.ColorSignal) color.RGBA {
	// we're usng the ColorSignal to index an array so we need to be extra
	// careful to make sure the value is valid. if it's not a valid index then
	// assume the intention was video black
	if col == signal.VideoBlack {
		return VideoBlack
	}
	return spec.Colors[col]
}

// From the Stella Programmer's Guide:
//
// "Each scan lines starts with 68 clock counts of horizontal blank (not seen on
// the TV screen) followed by 160 clock counts to fully scan one line of TV
// picture. When the electron beam reaches the end of a scan line, it returns
// to the left side of the screen, waits for the 68 horizontal blank clock
// counts, and proceeds to draw the next line below."
//
// Clock counts are the same for both TV specifications. Vertical information should
// be accessed via SpecNTSC or SpecPAL
const (
	ClksHBlank   = 68
	ClksVisible  = 160
	ClksScanline = 228
)

// The absolute number of scanlines allowed by the TV regardless of the current
// specification. The value here is arbitrary but it represents the natural
// resonance of the vertical oscilator
//
// For reference, this is equivalent to a frequency of approximately 50Hz (the
// precise frequency depends on the exact TV specification)
const AbsoluteMaxScanlines = 313

// The absolute number of color clock allowed by the TV regardless of
// specification
const AbsoluteMaxClks = AbsoluteMaxScanlines * ClksScanline

// The number of scanlines at which to flip between the NTSC and PAL
// specifications. If the number of scanlines generated is greater than this
// value then the PAL specification should be assumed
const PALTrigger = 302

// AspectBias transforms the scaling factor for the X axis. in other words,
// for width of every pixel is height of every pixel multiplied by the
// aspect bias
//
// Earlier versions of the emulator set this according to the specification that
// was in use. However, I now believe this is wrong and a nominal value of 0.91
// is good for all specifications. For comparison, the historical value for PAL
// output was set to 1.09
const AspectBias = 0.91

// SpecNTSC is the specification for NTSC television type
var SpecNTSC Spec

// SpecPAL is the specification for PAL television type
var SpecPAL Spec

// SpecPAL_M is the specification for PALM television type
var SpecPAL_M Spec

// SpecSECAM is the specification for SECAM television type
var SpecSECAM Spec

func init() {
	SpecNTSC = Spec{
		ID:                 "NTSC",
		HorizontalScanRate: 15734.26,
		Colors:             PaletteNTSC,
		ScanlinesVSync:     3,
		ScanlinesVBlank:    37,
		ScanlinesVisible:   192,
		ScanlinesOverscan:  30,
		ScanlinesTotal:     262,
		RefreshRate:        60.0,
	}
	SpecNTSC.RefreshRate = SpecNTSC.HorizontalScanRate / float32(SpecNTSC.ScanlinesTotal)
	SpecNTSC.AtariSafeVisibleTop = SpecNTSC.ScanlinesVBlank + SpecNTSC.ScanlinesVSync
	SpecNTSC.AtariSafeVisibleBottom = SpecNTSC.ScanlinesTotal - SpecNTSC.ScanlinesOverscan

	SpecPAL = Spec{
		ID:                 "PAL",
		HorizontalScanRate: 15625.00,
		Colors:             PalettePAL,
		ScanlinesVSync:     3,
		ScanlinesVBlank:    45,
		ScanlinesVisible:   228,
		ScanlinesOverscan:  36,
		ScanlinesTotal:     312,
		RefreshRate:        50.0,
	}

	SpecPAL.RefreshRate = SpecPAL.HorizontalScanRate / float32(SpecPAL.ScanlinesTotal)
	SpecPAL.AtariSafeVisibleTop = SpecPAL.ScanlinesVBlank + SpecPAL.ScanlinesVSync
	SpecPAL.AtariSafeVisibleBottom = SpecPAL.ScanlinesTotal - SpecPAL.ScanlinesOverscan

	SpecPAL_M = Spec{
		ID:                 "PAL-M",
		HorizontalScanRate: 15734.26,
		Colors:             PaletteNTSC,
		ScanlinesVSync:     3,
		ScanlinesVBlank:    37,
		ScanlinesVisible:   192,
		ScanlinesOverscan:  30,
		ScanlinesTotal:     262,
		RefreshRate:        60.0,
	}

	SpecPAL_M.RefreshRate = SpecPAL_M.HorizontalScanRate / float32(SpecPAL_M.ScanlinesTotal)
	SpecPAL_M.AtariSafeVisibleTop = SpecPAL_M.ScanlinesVBlank + SpecPAL_M.ScanlinesVSync
	SpecPAL_M.AtariSafeVisibleBottom = SpecPAL_M.ScanlinesTotal - SpecPAL_M.ScanlinesOverscan

	SpecSECAM = Spec{
		ID:                 "SECAM",
		HorizontalScanRate: 15625.00,
		Colors:             PaletteSECAM,
		ScanlinesVSync:     3,
		ScanlinesVBlank:    45,
		ScanlinesVisible:   228,
		ScanlinesOverscan:  36,
		ScanlinesTotal:     312,
		RefreshRate:        50.0,
	}

	SpecSECAM.RefreshRate = SpecSECAM.HorizontalScanRate / float32(SpecSECAM.ScanlinesTotal)
	SpecSECAM.AtariSafeVisibleTop = SpecSECAM.ScanlinesVBlank + SpecSECAM.ScanlinesVSync
	SpecSECAM.AtariSafeVisibleBottom = SpecSECAM.ScanlinesTotal - SpecSECAM.ScanlinesOverscan

	// Extended values:
	// - Spike's Peak likes a bottom scanline of 250 (NTSC). this is the largest requirement I've seen.
	SpecNTSC.NewSafeVisibleTop = 23
	SpecNTSC.NewSafeVisibleBottom = 250
	SpecPAL.NewSafeVisibleTop = 30
	SpecPAL.NewSafeVisibleBottom = 299
	SpecPAL_M.NewSafeVisibleTop = 20
	SpecPAL_M.NewSafeVisibleBottom = 249
	SpecSECAM.NewSafeVisibleTop = 30
	SpecSECAM.NewSafeVisibleBottom = 299
}
