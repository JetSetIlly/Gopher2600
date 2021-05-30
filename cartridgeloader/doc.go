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

// Package cartridgeloader is used to specify the cartridge data that is to be
// attached to the emulated VCS.
//
// When the cartridge is ready to be loaded into the emulator, the Load()
// function should be used. The Load() function handles loading of data from a
// different sources. Currently on local-file and data over HTTP is supported.
//
// As well as the filename, the Loader type allows the cartridge mapping to be
// specified, if required. The simplest instantiation therefore is:
//
//     cl := cartridgeloader.Loader{
//             Filename: "roms/Pitfall.bin",
//     }
//
// It is preferred however, that the NewLoader() function is used to initialise
// the Loader.
//
// The NewLoader() function accepts two arguments. The filename of the
// cartridge (which might be a HTTP url) and the cartridge mapper. In most
// cases the mapper will be "AUTO" to indicate that we don't know (or
// particular care) what the mapping format is.
//
// File Extensions
//
// The file extension of a file will specify the cartridge mapping and will
// cause the emulation to use that mapping. Most 2600 ROM files have the
// extension "bin" but sometimes it is necessary to specify explicitly what
// the mapper is.
//
// The following quoted file extensions are recognised (case insenstive):
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
//	Supercharger	"AR"
//	DF				"DF"
//	3E				"3E"
//	3E+				"3E+"
//	Superbank		"SB"
//	DPC (Pitfall2)  "DPC"
//	DPC+			"DP+"
//	CDF				"CDF" (including CDFJ)
//
// In addition to these above there a three "special" mapping formats. Which
// aren't really mappers in the normal sense but which will none-the-less
// trigger a particular cartridge emulation path.
//
// "MP3" and "WAV" indicate that the emulation should use the Supercharger
// mapper but with the understanding that the data will be loaded from audio
// data.
//
// "MVC" indicates that the data is a MovieCart stream. Because of the
// potential size of these files, data is streamed from the file. Also,
// steaming over HTTP is not yet currently supported.
//
// Finally a file extension of "BIN", "ROM", "A26" will tell the mapping system
// to "fingerprint" the cartridge data to ascertain the mapping type.
//
// Fingerprinting is not handled by the cartridlgeloader package.
//
// Hash
//
// The Hash field of the Loader type contains the SHA1 value of the loaded
// data. It is valid after the Load() function has completed successfully. If
// the field is not empty before Load() is called, that value will be compared
// with the calculated value. An error is returned if the values differ.
//
// In most cases you wouldn't want to change the Hash field before calling the
// Load() function but it is sometimes useful to make sure the correct file is
// being loaded - for example the recorder package uses it to help make sure
// playback is correct.
//
// The hash value is invalid/unused in the case of streamed data
//
// Streaming
//
// For some cartridge types it is necessary to stream bytes from the file
// rather than load them all at once. For these types of cartridges the Load()
// function will open a stream and readbable via the StreamedData field.
//
// The function IsStreaming() returns true if data is to be read from the
// StreamedData field rather than the Data field.
//
// For streaming to work NewLoader() must have been used to instantiate the
// Loader type.
//
// Closing
//
// Instances of Loader must be closed with Close() when it is no longer
// required.
//
// OnInserted
//
// The OnInserted field is used by some cartridges to indicate when the
// cartridge data has been loaded into memory and something else needs to
// happen that is outside of the scope of the cartridge package. Currently,
// this is used by the Supercharder and PlusROM.
//
package cartridgeloader
