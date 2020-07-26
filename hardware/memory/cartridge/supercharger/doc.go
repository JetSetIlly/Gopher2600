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

// Package supercharger implements the tape based cartridge format. The
// implementation is complex enough for it to be spread over more than one file
// and so for purposes of clarity it has been placed in its own package.
//
// The supercharger is interfaced in the same way as other cartridge formats.
// The fastload mechanism however, requires special handling, which is
// unfortunate but unavoidable. It is worth summarising here:
//
// As a result of a fast-load of a BIN file into supercharger memory, the
// supercharger package will return an error of type supercharger.FastLoaded
//
// The FastLoaded error indicates an exception to the normal running of the
// emulation. For the loading to complete the error needs special handling.
// There are two places in Gopher2600 where this handling takes place. One is
// in the debugger.inputLoop() and the other in hardware.run() (the latter is
// called from the playmode package).
//
// Wherever it is handled, the error should be caught and interpreted as a
// function and called, with a reference to the emulator's CPU, RAM and Timer.
package supercharger
