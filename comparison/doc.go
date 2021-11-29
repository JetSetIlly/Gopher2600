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
// The package synchronises the two emulations and the main emulation (which
// we'll refer to as the "driver emulation") will always be one frame ahead of
// the comparison emulation. Either emulation will be stalled for the duration
// that the other emulation completes the next frame.
//
// User input is synchronised by setting the driver emulation's RIOT Ports as a
// driver and the comparison emulation's as a passenger (see RIOT package).
//
// Note that the driver emulation will be stalled and will not be able to service
// any of the normal communication channels for the duration that the
// comparison emulation is running.
//
// The comparison emulation will produce two streams of pixels. The first is
// the frame-by-frame video output of the emulation; and the second stream
// shows the differences (as white pixels) between corresponding frames from
// the two emulations. Each video stream has a one frame buffer.
//
// The comparison emulation does not handle the rewind state at all. This means
// that if the driver emulation is put into the rewinding state the constraints
// on how the emulations are synchronised will very likely be broken. For
// simplicity the comparison emulation should be abandoned whenever the driver
// emulation enters the rewinding state.
package comparison
