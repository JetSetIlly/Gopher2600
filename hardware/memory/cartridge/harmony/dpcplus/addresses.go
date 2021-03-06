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

import "github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony/arm7tdmi"

// the memory addresses from the point of view of the ARM processor.
const (
	driverOriginROM = arm7tdmi.FlashOrigin
	driverMemtopROM = arm7tdmi.FlashOrigin | 0x00000bff

	customOriginROM = arm7tdmi.FlashOrigin | 0x00000c00
	customMemtopROM = arm7tdmi.FlashOrigin | 0x00006bff

	dataOriginROM = arm7tdmi.FlashOrigin | 0x00006c00
	dataMemtopROM = arm7tdmi.FlashOrigin | 0x00007bff

	freqOriginROM = arm7tdmi.FlashOrigin | 0x00007c00
	freqMemtopROM = arm7tdmi.FlashOrigin | 0x00008000

	driverOriginRAM = arm7tdmi.SRAMOrigin | 0x00000000
	driverMemtopRAM = arm7tdmi.SRAMOrigin | 0x00000bff

	dataOriginRAM = arm7tdmi.SRAMOrigin | 0x00000c00
	dataMemtopRAM = arm7tdmi.SRAMOrigin | 0x00001bff

	freqOriginRAM = arm7tdmi.SRAMOrigin | 0x00001c00
	freqMemtopRAM = arm7tdmi.SRAMOrigin | 0x00002000

	// stack should be within the range of the RAM copy of the frequency tables.
	stackOriginRAM = arm7tdmi.SRAMOrigin | 0x00001fdc
)
