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

package arm

import "github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/peripherals"

type peripheral interface {
	Reset()
}

type peripheralMemory interface {
	Write(addr uint32, val uint32) (bool, string)
	Read(addr uint32) (uint32, bool, string)
}

type timer interface {
	Step(cycles uint32)
}

func (arm *ARM) addPeripherals() {
	if arm.mmap.HasRNG {
		rng := peripherals.NewRNG(arm.mmap)
		arm.state.peripherals = append(arm.state.peripherals, rng)
		arm.state.peripheralsMemory = append(arm.state.peripheralsMemory, rng)
	}
	if arm.mmap.HasTIMER {
		timer := peripherals.NewTimer(arm.mmap)
		arm.state.peripherals = append(arm.state.peripherals, timer)
		arm.state.peripheralsMemory = append(arm.state.peripheralsMemory, timer)
		arm.state.timers = append(arm.state.timers, timer)
	}
	if arm.mmap.HasTIM2 {
		timer2 := peripherals.NewTimer2(arm.mmap)
		arm.state.peripherals = append(arm.state.peripherals, timer2)
		arm.state.peripheralsMemory = append(arm.state.peripheralsMemory, timer2)
		arm.state.timers = append(arm.state.timers, timer2)
	}
}
