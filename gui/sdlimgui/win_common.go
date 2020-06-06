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

package sdlimgui

import "github.com/inkyblackness/imgui-go/v2"

// this file contains some useful emeddable types for implementations of the
// managedWindow interface

// windowManagement can be embedded into a real window struct for
// basic window management functionality. it partially implements the
// managedWindow interface.
type windowManagement struct {
	// prefer use of isOpen()/setOpen() instead of accessing the open field
	// directly
	open bool
}

func (wm *windowManagement) isOpen() bool {
	return wm.open
}

func (wm *windowManagement) setOpen(open bool) {
	wm.open = open
}

// widgetDimensions can be embedded in window structs that make use of
// precalculated widget dimensions of some common types
type widgetDimensions struct {
	twoDigitDim   imgui.Vec2
	threeDigitDim imgui.Vec2
	fourDigitDim  imgui.Vec2
	eightDigitDim imgui.Vec2
}

// window types that embed widgetDimensions should call this init() from the
// their own init() function
func (wd *widgetDimensions) init() {
	wd.twoDigitDim = imguiGetFrameDim("FF")
	wd.threeDigitDim = imguiGetFrameDim("FFF")
	wd.fourDigitDim = imguiGetFrameDim("FFFF")
	wd.eightDigitDim = imguiGetFrameDim("FFFFFFFF")
}
