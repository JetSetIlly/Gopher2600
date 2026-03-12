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

// Package supercharger implements the tape based cartridge format.
//
// The package supports both loading from a sound file (supporting most WAV and
// MP3 files) or from a "fastload" file.
//
// Tape loading "events" are handled through the notifications.Notify interface.
//
// When loading from a sound file, Supercharger events can be ignored if so
// desired but for fastload files, the emulator needs to help the Supercharger
// mapper. See the playmode package reference implementation for details.
//
// Mutliload tapes are supported from both sound file and fastload binaries. In
// the case of sound files the audio must be in a single file.
//
// Information about supercharger technology is found in the "sctech.txt" document.
//
// https://web.archive.org/web/19990210092458/https://www.primenet.com/~nickb/sctech.txt
//
// And for the fastload format, the best information is an old biglist email
// from Eckhard Stolberg. This document will be referred to as 'Stolberg' in any
// code comments.
//
// https://www.biglist.com/lists/stella/archives/199901/msg00026.html
package supercharger
