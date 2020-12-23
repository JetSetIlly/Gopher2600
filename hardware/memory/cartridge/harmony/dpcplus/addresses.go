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

package dpcplus

// the memory addresses from the point of view of the ARM processor.
const (
	driverOriginROM = 0x00000000
	driverMemtopROM = 0x00000bff

	customOriginROM = 0x00000c00
	customMemtopROM = 0x00006bff

	dataOriginROM = 0x00006c00
	dataMemtopROM = 0x00007bff

	freqOriginROM = 0x00007c00
	freqMemtopROM = 0x00008000

	driverOriginRAM = 0x40000000
	driverMemtopRAM = 0x40000bff

	dataOriginRAM = 0x40000c00
	dataMemtopRAM = 0x40001bff

	freqOriginRAM = 0x40001c00
	freqMemtopRAM = 0x40002000

	// stack should be within the range of the RAM copy of the frequency tables
	stackOriginRAM = 0x40001fdc
)
