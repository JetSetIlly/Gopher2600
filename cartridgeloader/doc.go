// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the gnu general public license as published by
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

// Package cartridgeloader is used to load cartridge data so that it can be used
// with the cartridge pacakage
//
// # File Extensions
//
// The file extension of a file will specify the cartridge mapping and will
// cause the emulation to use that mapping.
//
// The following file extensions are recognised and force the use of the
// specified mapper:
//
//	Atari 2k		"2k"
//	Atari 4k		"4k"
//	Atari 8k		"F8"
//	Atari 16k		"F6"
//	Atari 32k		"F4"
//	Atari 2k (RAM)	"2k+"
//	Atari 4k (RAM)	"4k+"
//	Atari 8k (RAM)	"F8+"
//	Atari 16k (RAM)	"F6+"
//	Atari 32k (RAM)	"F4+"
//	CBS				"FA"
//	Parker Bros		"E0"
//	M-Network		"E7"
//	Tigervision		"3F"
//	Supercharger	"AR", "MP3, "WAV"
//	DF				"DF"
//	3E				"3E"
//	3E+				"3E+"
//	Superbank		"SB"
//	DPC (Pitfall2)  "DPC"
//	DPC+			"DP+"
//	CDF				"CDF" (including CDFJ)
//	MovieCart		"MVC"
//
// File extensions are case insensitive.
//
// A file extension of "BIN", "ROM", "A26" indicates that the data should be
// fingerprinted as normal.
//
// # Hashes
//
// Creating a cartridge loader with NewLoaderFromFilename() or
// NewLoaderFromData() will also create a SHA1 and MD5 hash of the data. The
// amount of data used to create the has is limited to 1MB. For most cartridges
// this will mean the hash is taken using all the data but some cartridge are
// likely to have much more data than that.
package cartridgeloader
