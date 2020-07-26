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

// Package sdldebug implements the GUI interface. Suitable for applications
// that require a screen and debugging overlays. Currently, it can show:
//
//	- alternative "debug" colours
//	- show a "meta-pixel" overlay (see debugger.reflection package)
//	- show an unmasked screen, showing off-screen sprite pixels when using debug colors
package sdldebug
