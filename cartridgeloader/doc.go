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
//	Atari 2k        "2k"
//	Atari 4k        "4k"
//	Atari 8k        "F8"
//	Atari 16k       "F6"
//	Atari 32k       "F4"
//	Atari 2k (RAM)  "2k+"
//	Atari 4k (RAM)  "4k+"
//	Atari 8k (RAM)  "F8+"
//	Atari 16k (RAM) "F6+"
//	Atari 32k (RAM) "F4+"
//	CBS             "FA"
//	Parker Bros     "E0"
//	M-Network       "E7"
//	Tigervision     "3F"
//	Supercharger    "AR", "MP3, "WAV"
//	DF              "DF"
//	3E              "3E"
//	3E+             "3E+"
//	Superbank       "SB"
//	DPC (Pitfall2)  "DPC"
//	DPC+            "DP+"
//	CDF             "CDF" (including CDFJ)
//	MovieCart       "MVC"
//
// File extensions are case insensitive.
//
// A file extension of "BIN", "ROM", "A26" indicates that the data should be
// fingerprinted as normal.
//
// # Preloaded data
//
// Cartridges with lots of data wil be streamed off disk as required. For
// example, Moviecart or Supercharge audio tapes can be large and don't need to
// exist in memory for a very long time.
//
// However, for practical reasons the first 1MB of data of any file will be
// 'preloaded'. When reading cartridge data you don't need to worry about
// whether data has been preloaded or not, except that it does affect both
// hashing and fingerprinting.
//
// # Hashes
//
// The creation of a cartridge loader includes the creation of both a SHA1 and
// an MD5 hash. Hashes are useful for matching cartridges regardless of path
// or filename
//
// The data used to create the hash is limited to the data that has been
// preloaded (see above).
//
// # Fingerprinting
//
// Cartridge data can be checked for 'fingerprint' data that can be used to
// decide on the 'mapping' the cartridge uses. The three cartridge loader
// functions, Contains(), ContainsLimit() and Count() can be used to search the
// preloaded data (see above) for specific bytes sequences.
//
// More complex fingerprinting can be done with the Read() function. However,
// because the Read() function works with the complete cartridge and not just
// the preloaded data, care should be taken not to read too much of the data for
// reasons of computation time. The constant value FingerprintLimit is provided
// as a useful value to which a Read() loop can be limited.
//
// Once fingerprinting has been completed it is very important to remember to
// reset the Read() position with the Seek() command:
//
//	cartridgeloader.Seek(0, io.SeekStart)
package cartridgeloader
