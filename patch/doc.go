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

// Package patch is used to patch the contents of a cartridge. It works on
// cartridge memory once a cartridge file has been attached. The package does
// not implement the patching directly, rather the different cartridge mappers
// (see cartridge package) deal with that individually.
//
// This package simply loads the patch instructions, interprets them and calls
// the cartridge.Patch() function.
//
// The first format is simplt the output of "cmp -l <old_file> <new_file>", an
// example of which is shown below:
//
//	 862  22 200
//	 863 360 376
//	3713 377 242
//	3715 377 232
//	3716 377 114
//	3717 377  22
//	3718 377 360
//
// The first column is the offset, expressed as a decimal number and measure
// from one. The second column is the current value of the offset being
// changed, and the third column is the value it is being changed to. The
// values in column 2 and three are expressed in octal!
//
// The second format is what I have called the "neo" format. This seems to be
// an ad-hoc format taken from the "In case you can't wait" section of the
// following web page (the domain of which was used to help name the format):
//
//	"Fixing E.T. The Extra-Terrestrial for the Atari 2600"
//
//	http://www.neocomputer.org/projects/et/
//
// The following extract illustrates the format:
//
//	-------------------------------------------
//	- E.T. is Not Green
//	-------------------------------------------
//	17FA: FE FC F8 F8 F8
//	1DE8: 04
//
// Rules:
//
//  1. Lines beginning with a hyphen or white space are ignored
//  2. Offset and values are expressed in hex (case-insensitive)
//  3. Values and offsets are separated by a colon
//  4. Multiple values on a line are poked into consecutive offsets, starting
//     from the offset value
//
// Note that offsets are expressed with origin zero and have no relationship
// to how memory is mapped inside the VCS. Imagine that the patches are being
// applied to the cartridge file image. The cartridge mapper handles the VCS
// memory side of things.
package patch
