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

// Package cartridge fully implements loading of mapping of cartridge memory.
// Cartridge memory is memory that is peripheral to the VCS and can grow quite
// complex.
//
// There are many different types of mapping scheme supported by the package.
//
// Some cartridge types contain additional RAM but the main difference is how
// they map additional ROM into the relatively small address space available
// for cartridges in the VCS. This is called bank-switching. All of these
// differences are handled transparently by the package.
//
// Currently supported cartridge types are listed below. The strings in
// quotation marks are the identifiers that should be used to specify a
// particular mapping in the Mapping field of cartridgeloader.Loader. An empty
// string or "AUTO" tells the cartridge system to make a best guess.
//
//	Atari 2k		"2k"
//	Atari 4k		"4k"
//	Atari 8k		"F8"
//	Atari 16k		"F6"
//	Atari 32k		"F4"
//	CBS case		"FA"
//	M-Network		"E7"
//	Parker Bros		"E0"
//	Tigervision		"3F"
//	DPC (Pitfall2)  "DPC"
//	DPC+			"DPC+"
//	3E+				"3E+"
//	Supercharger	"AR"
package cartridge
