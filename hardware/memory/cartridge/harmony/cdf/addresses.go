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
// the memory addresses from the point of view of the ARM processor.

package cdf

const (
	driverOriginROM = 0x00000000
	driverMemtopROM = 0x000007ff

	customOriginROM = 0x00000800
	customMemtopROM = 0x00007fff

	driverOriginRAM = 0x40000000
	driverMemtopRAM = 0x400007ff

	dataOriginRAM = 0x40000800
	dataMemtopRAM = 0x400017ff

	variablesOriginRAM = 0x40001800
	variablesMemtopRAM = 0x40001fff

	// stack should be within the range of the RAM copy of the variables
	stackOriginRAM = 0x40001fdc
)
