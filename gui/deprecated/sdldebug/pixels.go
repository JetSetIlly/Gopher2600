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

package sdldebug

const pixelDepth = 4
const pixelWidth = 2.0

// the pixels type stores all pixel information for the screen.
//
// the overlay type looks after it's own pixels.
type pixels struct {
	l       int
	regular []byte
	dbg     []byte
	clr     []byte
}

// create a new instance of the pixels type. called everytime the screen
// dimensions change.
func newPixels(w, h int) *pixels {
	l := w * h * pixelDepth

	pxl := &pixels{
		l:       l,
		regular: make([]byte, l),
		dbg:     make([]byte, l),
		clr:     make([]byte, l),
	}

	// set alpha bit for regular and dbg pixels to opaque. we'll be changing
	// this value during clear() and setPixel() operations but it's important
	// we set it to opaque for when we first use the pixels, or we'll get to
	// see nasty artefacts on the screen.
	for i := pixelDepth - 1; i < l; i += pixelDepth {
		pxl.regular[i] = 255
		pxl.dbg[i] = 255
	}

	return pxl
}

func (pxl pixels) length() int {
	return pxl.l
}

func (pxl pixels) clear() {
	copy(pxl.regular, pxl.clr)
	copy(pxl.dbg, pxl.clr)
}
