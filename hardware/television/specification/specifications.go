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

	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// SpecList is the list of specifications that the television may adopt.
var SpecList = []string{"NTSC", "PAL"}

// Spec is used to define the two television specifications.
type Spec struct {
	ID     string
	Colors []color.RGBA

	// the number of scanlines the 2600 Programmer's guide recommends for the
	// top/bottom parts of the screen:
	//
	// "A typical frame will consists of 3 vertical sync (VSYNC) lines*, 37 vertical
	// blank (VBLANK) lines, 192 TV picture lines, and 30 overscan lines. Atari’s
	// research has shown that this pattern will work on all types of TV sets."
	//
	// the above figures are in reference to the NTSC protocol
	ScanlinesVSync    int
	scanlinesVBlank   int
	ScanlinesVisible  int
	ScanlinesOverscan int

	// the total number of scanlines for the entire frame is the sum of the
	// four individual portions
	ScanlinesTotal int

	// the scanline at which the VBLANK should be turned off (Top) and
	// turned back on again (Bottom). the period between the top and bottom
	// scanline is the visible portion of the screen.
	//
	// in practice, the VCS can turn VBLANK on and off at any time; what the
	// two values below represent what "Atari's research" has shown to be safe.
	// by definition this means that:
	//
	//	Top = VSync + Vblank
	//
	//	Bottom = Top + Visible
	//
	// or
	//
	//	Bottom = Total - Overscan
	ScanlineTop    int
	ScanlineBottom int

	// AspectBias transforms the scaling factor for the X axis. in other words,
	// for width of every pixel is height of every pixel multiplied by the
	// aspect bias

	// AaspectBias transforms the scaling factor for the X axis.
	// values taken from Stella emualtor. useful for A/B testing
	AspectBias float32

	// the number of frames per second required by the specification
	FramesPerSecond float32

	// if the generated image is exactly ScanlinesTotal in height then how many
	// pixels would that be. used for frame rate measurement.
	IdealPixelsPerFrame int
}

// GetColor translates a signals to the color type.
func (spec *Spec) GetColor(col signal.ColorSignal) color.RGBA {
	// we're usng the ColorSignal to index an array so we need to be extra
	// careful to make sure the value is valid. if it's not a valid index then
	// assume the intention was video black
	if col == signal.VideoBlack {
		return videoBlack
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
// Horizontal clock counts are the same for both TV specifications. Vertical
// information should be accessed via SpecNTSC or SpecPAL.
const (
	HorizClksHBlank   = 68
	HorizClksVisible  = 160
	HorizClksScanline = 228
)

// SpecNTSC is the specification for NTSC television types.
var SpecNTSC Spec

// SpecPAL is the specification for PAL television types.
var SpecPAL Spec

func init() {
	SpecNTSC = Spec{
		ID:                "NTSC",
		Colors:            PaletteNTSC,
		ScanlinesVSync:    3,
		scanlinesVBlank:   37,
		ScanlinesVisible:  192,
		ScanlinesOverscan: 30,
		ScanlinesTotal:    262,
		FramesPerSecond:   60.0,
		AspectBias:        0.91,
	}

	SpecNTSC.ScanlineTop = SpecNTSC.scanlinesVBlank + SpecNTSC.ScanlinesVSync
	SpecNTSC.ScanlineBottom = SpecNTSC.ScanlinesTotal - SpecNTSC.ScanlinesOverscan
	SpecNTSC.IdealPixelsPerFrame = SpecNTSC.ScanlinesTotal * HorizClksScanline

	SpecPAL = Spec{
		ID:                "PAL",
		Colors:            PalettePAL,
		ScanlinesVSync:    3,
		scanlinesVBlank:   45,
		ScanlinesVisible:  228,
		ScanlinesOverscan: 36,
		ScanlinesTotal:    312,
		FramesPerSecond:   50.0,
		AspectBias:        1.09,
	}

	SpecPAL.ScanlineTop = SpecPAL.scanlinesVBlank + SpecPAL.ScanlinesVSync
	SpecPAL.ScanlineBottom = SpecPAL.ScanlinesTotal - SpecPAL.ScanlinesOverscan
	SpecNTSC.IdealPixelsPerFrame = SpecPAL.ScanlinesTotal * HorizClksScanline
}
