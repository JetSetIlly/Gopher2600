// this file is part of gopher2600.
//
// gopher2600 is free software: you can redistribute it and/or modify
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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// Package cartridgeloader represents cartridge data when not attached to the
// VCS. When a reference to a cartridge is required functions expect an
// instance of cartridgeloader.Loader.
//
//	cl := cartridgeloader.Loader{
//		Filename: "roms/Pitfall.bin",
//	}
//
// When the cartridge is ready to be loaded the emulator calls the Load()
// function. This function currently handles files (specified with Filename)
// that are stored locally and also over http. Other protocols could easily be
// added. A good improvement would be to allow loading from zip or tar files.
package cartridgeloader
