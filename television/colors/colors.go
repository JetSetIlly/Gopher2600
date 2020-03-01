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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package colors

// RGB represents colors as 3-tuple of bytes
type RGB struct {
	Red   byte
	Green byte
	Blue  byte
}

// Palette is a collection of colours
type Palette []RGB

// PaletteNTSC is the collection of NTSC colours
var PaletteNTSC = Palette{}

// PalettePAL is the collection of PAL colours
var PalettePAL = Palette{}

// PaletteAlt is the collection of ALT colours
var PaletteAlt = Palette{}

// AltColor is the alternative color for each pixel.
type AltColor int

// VideoBlack is the color produced by a television in the absence of a color
// signal
var VideoBlack = RGB{0, 0, 0}

// List of valid AltColorSignals
const (
	AltColBackground AltColor = iota
	AltColBall
	AltColPlayfield
	AltColPlayer0
	AltColPlayer1
	AltColMissile0
	AltColMissile1
	altColCount
)

// GetAltColor translates an alternative color signals to the color type
func GetAltColor(altCol AltColor) RGB {
	return PaletteAlt[altCol]
}
