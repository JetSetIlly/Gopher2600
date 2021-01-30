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

import "github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony/arm7tdmi"

const (
	driverOriginROM = arm7tdmi.FlashOrigin
	driverMemtopROM = arm7tdmi.FlashOrigin | 0x000007ff

	customOriginROM = arm7tdmi.FlashOrigin | 0x00000800
	customMemtopROM = arm7tdmi.Flash32kMemtop

	driverOriginRAM = arm7tdmi.SRAMOrigin
	driverMemtopRAM = arm7tdmi.SRAMOrigin | 0x000007ff

	dataOriginRAM = arm7tdmi.SRAMOrigin | 0x00000800
	dataMemtopRAM = arm7tdmi.SRAMOrigin | 0x000017ff

	variablesOriginRAM = arm7tdmi.SRAMOrigin | 0x00001800
	variablesMemtopRAM = arm7tdmi.SRAMOrigin | 0x00001fff

	// stack should be within the range of the RAM copy of the variables.
	stackOriginRAM = arm7tdmi.SRAMOrigin | 0x00001fdc
)
