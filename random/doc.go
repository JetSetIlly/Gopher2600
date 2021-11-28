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
// It ultimately uses the RNG from the standard library but adds an
// understanding of how time is measured inside the emulation and ensures that
// the same random number is generated at the same point in the emulation's
// "timeline".
//
// An essential ingredient when generating random numbers in parallel
// emulations, that are meant to be identical; and when rewinding and replaying
// a single emulation.
package random
