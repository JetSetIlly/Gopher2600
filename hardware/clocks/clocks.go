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

// Package clocks defines the constant values that define the speed of the main
// clock in the VCS console.
//
// In addition to the clock value in the VCS type, the constant values are also
// used for colour generation.
//
// It should also used maybe, for the Supercharger soundloading. However, a
// choice has been made not to complicate the soundload code because it doesn't
// seem to make a difference to loading effectivenss
//
// Values taken from:
// http://www.taswegian.com/WoodgrainWizard/tiki-index.php?page=Clock-Speeds
package clocks

const (
	NTSC  = 1.193182
	PAL   = 1.182298
	PAL_M = 1.191870
	SECAM = 1.187500
)

const (
	NTSC_TIA  = NTSC * 3
	PAL_TIA   = PAL * 3
	PAL_M_TIA = PAL_M * 3
	SECAM_TIA = SECAM * 3
)
