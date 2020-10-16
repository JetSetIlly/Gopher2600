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

package reflection

import (
	"image/color"
)

// PaletteElements lists the colors to be used when displaying TIA video in a
// debugger's "debug colors" mode. The default colors are the same as the the
// debug colors found in the Stella emulator.
var PaletteElements = []color.RGBA{
	{R: 17, G: 17, B: 17, A: 255},
	{R: 132, G: 200, B: 252, A: 255},
	{R: 146, G: 70, B: 192, A: 255},
	{R: 144, G: 28, B: 0, A: 255},
	{R: 232, G: 232, B: 74, A: 255},
	{R: 213, G: 130, B: 74, A: 255},
	{R: 50, G: 132, B: 50, A: 255},
}

// PaletteEvents lists the colors to be used for reflected events. For example,
// when WSYNC is active the PaletteEvent["WSYNC"] entry should be used.
var PaletteEvents = map[string]color.RGBA{
	"WSYNC":         {R: 50, G: 50, B: 255, A: 100},
	"Collisions":    {R: 255, G: 25, B: 25, A: 200},
	"HMOVE delay":   {R: 150, G: 50, B: 50, A: 150},
	"HMOVE":         {R: 50, G: 150, B: 50, A: 150},
	"HMOVE latched": {R: 50, G: 50, B: 150, A: 150},
	"Unchanged":     {R: 255, G: 100, B: 25, A: 150},
}
