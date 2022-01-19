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

// Package random should be used in preference to the math/rand package when a
// random number is required inside the emulation.
//
// There are two functions belonging to the Rewind type that return random
// numbers:
//
// Rewindable() returns numbers based on the current television coordinates.
// The number will always return the same number for the same coordinates. As
// such it is compatible with the emulator's rewind system.
//
// NoRewind() returns random numbers regardless of the current television
// coordinates. It is therefore, not compatiable with the emulator's rewind
// system.
//
// Parallel emulators should return the same sequence of random numbers even if
// NoRewind() is used.
//
// If the same random numbers are required every single time then set ZeroSeed
// to true. This is useful for testing purposes.
package random
