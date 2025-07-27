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

// Package random is used instead of the math/rand or math/rand/v2 packages when
// a random number is required inside the emulation.
//
// The Random type is an instance of the standard rand.Rand type and in
// addition, provides the Rewindable() function.
//
// The Rewindable() function returns numbers based on the current television
// coordinates. The number will always return the same number for the same
// coordinates. As such it is compatible with the emulator's rewind system.
//
// If predictable random numbers are required from the Rewindable() function
// then set ZeroSeed to true. This can be enabled and disabled at any time.
// Predicable random numbers is useful for testing purposes.
package random
