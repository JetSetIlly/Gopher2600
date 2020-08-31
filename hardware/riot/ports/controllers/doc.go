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

// Package controllers contains the implementations for all the emulated
// controllers for the VCS.
//
// The Auto type handles flipping of the other controller types according to
// user input and the state of the machine. The Auto type will forward all
// functions to the "real" controller (ie. the stick, paddle or keyboard)
// transparently. So for example, ID() will return the ID() of the "real"
// controller. If you really need to know whether the real controller has been
// automatically selected via the Auto type then you can (test the Player 0
// port, for example):
//
//	if _, ok := ports.Player0.(controllers.Auto); ok {
//		// is auto
//	} else {
//		// is not auto
//	}
//
package controllers
