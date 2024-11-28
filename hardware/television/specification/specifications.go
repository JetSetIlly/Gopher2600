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

	// trim file extension to avoid any remote possibility of a false match
	s = strings.TrimSuffix(s, filepath.Ext(s))

	s = strings.ToUpper(s)
	for _, spec := range ReqSpecList {
		// searching from the end because spec directives usually appear at the
		// end of a string
		n := strings.LastIndex(s, spec)

		// ignore any matches at the beginning of the filename to reduce
		// possibility of false matches
		//
		// for example, a ROM named "paletteTest.bin" will falsely say it is a
		// PAL ROM (because palette starts with pal)
		//
		// of course, "myPaletteTest.bin" will match and likely be a false match
		// but this will do for now. an improvement to this scheme would require
		// thinking about the surrounding characters; or more simply, to insist
		// on strict case matching
		if n > 0 {
			switch spec {
			case "AUTO":
				// ignore appearance of auto in the string
			case "PAL-60":
				id = "PAL60"
			default:
				id = spec
			}
			break // end for loop on the first match
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

	// the ideal visible top/bottom valuss are the inital values taken by the
	// resizer. in the case of PAL and SECAM these are the same as the Atari
	// Safe top/bottom values but in the case of NTSC and PAL_M the ideal values
	// create a slightly larger aperture in order to create a 4:3 image in the
	// majority of cases
	IdealVisibleTop    int
	IdealVisibleBottom int

	// resizing of the TV is problematic because we can't rely on the VBLANK to
	// tell us when the pixels are meant to be in view. The ExtendedVisibleTop an
	// ExtendedVisibleBottom are the min/max values that the resizer should allow.
	//
	// think of these as the "modern" safe values as compared to the Atari
	// defined safe values.
	ExtendedVisibleTop    int
	ExtendedVisibleBottom int
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

// These width and height values can be used to create a TV image (pixel buffer)
// of the appropriate asepect ratio. The values can be scaled as appropriate
const (
	WidthTV    = ClksVisible
	WidthHDTV  = ClksVisible
	HeightTV   = WidthTV * 3 / 4
	HeightHDTV = WidthHDTV * 9 / 16
)

// The absolute number of scanlines allowed by the TV regardless of the current
// specification. The value here is arbitrary but it represents the natural
// resonance of the vertical oscilator
//
// For reference, this is equivalent to a frequency of approximately 45Hz.
//
// Changing this value will likely affect any previously recorded hashes of the
// full screen. For example, playback recordings or the videochess bot
const AbsoluteMaxScanlines = 350

// The absolute number of color clock allowed by the TV regardless of
// specification
const AbsoluteMaxClks = AbsoluteMaxScanlines * ClksScanline

// The number of scanlines at which to flip between the NTSC and PAL
// specifications. If the number of scanlines generated is greater than this
// value then the PAL specification should be assumed
const PALTrigger = 302

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

	// ideal values:

	// NTSC AND PAL_M have been calculated by applying a 4:3 ratio to 160 (ClksVisible)
	//	 = 160 / 3 * 4 = 213.333
	//
	// the 'atari safe' visible is 192
	//   = 213.333 - 192 = 21.3333
	//
	// we therefore adjust the 'atari safe' top and bottom values by 11 and 10
	// to give us a nice 4:3 ratio
	//
	// if we don't do this then we can still fit the atari safe values into a
	// 4:3 aperture but many games will look stretched. far better to have
	// visible VBLANK
	SpecNTSC.IdealVisibleTop = SpecNTSC.AtariSafeVisibleTop - 11
	SpecNTSC.IdealVisibleBottom = SpecNTSC.AtariSafeVisibleBottom + 10
	SpecPAL_M.IdealVisibleTop = SpecPAL_M.AtariSafeVisibleTop - 11
	SpecPAL_M.IdealVisibleBottom = SpecPAL_M.AtariSafeVisibleBottom + 10

	// PAL and SECAM are the same as the atari safe values
	SpecPAL.IdealVisibleTop = SpecPAL.AtariSafeVisibleTop
	SpecPAL.IdealVisibleBottom = SpecPAL.AtariSafeVisibleBottom
	SpecSECAM.IdealVisibleTop = SpecPAL.AtariSafeVisibleTop
	SpecSECAM.IdealVisibleBottom = SpecPAL.AtariSafeVisibleBottom

	// extended values:
	// - NTSC: Spike's Peak likes a bottom scanline of 250
	// - PAL: Acid drop extends the main play area to around 305 scanlines
	SpecNTSC.ExtendedVisibleTop = 23
	SpecNTSC.ExtendedVisibleBottom = 250
	SpecPAL.ExtendedVisibleTop = 30
	SpecPAL.ExtendedVisibleBottom = 305
	SpecPAL_M.ExtendedVisibleTop = 20
	SpecPAL_M.ExtendedVisibleBottom = 249
	SpecSECAM.ExtendedVisibleTop = 30
	SpecSECAM.ExtendedVisibleBottom = 299
}
