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

// Package cartridge fully implements loading of mapping of cartridge memory.
//
// There are many different types of cartridge most of which are supported by
// the package. Some cartridge types contain additional RAM but the main
// difference is how they map additional ROM to the relatively small address
// space available for cartridges in the VCS. This is called bank-switching.
// All of these differences are handled transparently by the package.
//
// Currently supported cartridge types are:
//
//	- Atari 2k / 4k / 8k / 16k and 32k
//
//	- the above with additional Superchip (additional RAM in other words)
//
//	- Parker Bros.
//
//	- MNetwork
//
//	- Tigervision
//
//	- CBS
package cartridge
