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

// Package gui is an abstraction layer for real GUI implementations. It defines
// the Events that can be passed from the GUI to the emulation code and also
// the Requests that can be made from the emulation code to the GUI.
// Implementations need to convert their specific signals and requests to and
// from these abstractions.
//
// It also introduces the idea of metapixels. Metapixels can be thought of as
// supplementary signals to the underlying television. The GUI can then present
// the metapixels as an overlay. The ReqToggleOverlay request is intended for
// this.
package gui
