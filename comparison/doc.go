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

// Package comparison facilitates the running of a comparison emulator
// alongside the main emulation.
//
// The package synchronises the two emulations and the main emulation will
// always be exactly one frame ahead of the comparison emulation. Either
// emulation will be stalled for the duration that the other emulation
// completes the next frame.
//
// User input is synchronised by setting the main emulation's RIOT Ports as a
// driver and the comparison emulation's as a passenger (see RIOT package).
//
// Note that the main emulation will be stalled and will not be able to service
// any of the normal communication channels for the duration that the
// comparison emulation is running.
//
// The comparison emulation will produce two streams of pixels. The first is
// the frame-by-frame video output of the emulation; and the second stream
// shows the differences (as white pixels) between corresponding frames from
// the two emulations. Each video stream has a one frame buffer.
package comparison
