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

// Package cartridgeloader is used to specify the data that is to be attached
// to the emulated VCS.
//
// When the cartridge is ready to be loaded into the emulator, the Load()
// function should be used. The Load() function handles loading of data from a
// different sources. Currently on local-file and data over HTTP is supported.
//
// As well as the filename, the Loader type allows the cartridge mapping to be
// specified, if required.
//
// The simplest instance of the Loader type:
//
//	cl := cartridgeloader.Loader{
//		Filename: "roms/Pitfall.bin",
//	}
//
// It is preferred however that the NewLoader() function is used. The
// NewLoader() function will set the mapping field automatically according to
// the filename extension.
package cartridgeloader
