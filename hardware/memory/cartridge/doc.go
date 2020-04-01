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
// Currently supported cartridge types are listed below. The strings in
// quotation marks are the identifiers that should be used to specify a
// particular format in the Format field of cartridgeloader.Loader:
//
// Atari 2k			"2k"
// Atari 4k			"4k"
// Atari 8k			"F8"
// Atari 16k		"F6"
// Atari 32k		"F4"
// CBS case			"FA"
// M-Network		"E7"
// Parker Bros		"E0"
// Tigervision		"3F"
package cartridge
