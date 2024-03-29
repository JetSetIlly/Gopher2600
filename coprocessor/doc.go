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

// Package coprocessor contains the helper functions for cartridge
// coprocessors. In practice this means the ARM processor but we'll try to keep
// it general in case of future developments in the 2600 scene.
//
// The two subpackages, developer and disassembly, are distinct. The reason for
// the distinction is this: the developer package will only be used if the
// development files for the emulated ROM can be found. The disassembly package
// meanwhile, will work with any ROM and will endeavour to provide an accurate
// disassembly of the running coprocessor program.
package coprocessor
