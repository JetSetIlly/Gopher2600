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
// The package supports both loading from a sound file (supporting most WAV and
// MP3 files) or from a "fastload" file.
//
// Tape loading "events" are handled through the cartridgeloader packages
// VCSHook mechanism. See the mapper.Event type for list of Supercharger
// events.
//
// When loading from a sound file, Supercharger events can be ignored if so
// desired but for fastload files, the emulator needs to help the Supercharger
// mapper. See the playmode package reference implementation for details.
package supercharger
