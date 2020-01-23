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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package input

import (
	"fmt"
	"gopher2600/hardware/memory/bus"
)

// despite the placement of the input package in the source tree, input is
// actually handled in part by the TIA chip. as such, the hand controllers
// require access to TIA memory. the inputMemory type encapsulates both memory
// areas through the InputDeviceBus
type inputMemory struct {
	riot bus.InputDeviceBus
	tia  bus.InputDeviceBus
}

// Input implements the input/output part of the RIOT (the IO in RIOT)
type Input struct {
	mem inputMemory

	Panel           *Panel
	HandController0 *HandController
	HandController1 *HandController
}

// NewInput is the preferred method of initialisation of the Input type. Note
// that input devices require access to TIA memory as well as RIOT memory,
// breaking the abstraction somewhat, but it can't be helped. The NewInput()
// function therefore requires two arguments one to the RIOT chip bus and one
// to the TIA chip bus.
func NewInput(riotMem bus.ChipBus, tiaMem bus.ChipBus) (*Input, error) {
	inp := &Input{
		mem: inputMemory{
			riot: riotMem.(bus.InputDeviceBus),
			tia:  tiaMem.(bus.InputDeviceBus),
		},
	}

	inp.Panel = NewPanel(&inp.mem)
	if inp.Panel == nil {
		return nil, fmt.Errorf("can't create control panel")
	}

	inp.HandController0 = NewHandController0(&inp.mem)
	if inp.HandController0 == nil {
		return nil, fmt.Errorf("can't create player 0 port")
	}

	inp.HandController1 = NewHandController1(&inp.mem)
	if inp.HandController1 == nil {
		return nil, fmt.Errorf("can't create player 1 port")
	}

	return inp, nil
}

// ReadMemory checks to see if ChipData applies to the Input type and
// updates the internal controller/panel states accordingly. Returns true if
// the ChipData was *not* serviced.
func (inp *Input) ReadMemory(data bus.ChipData) bool {
	switch data.Name {
	case "SWCHA":
	case "SWACNT":
		inp.HandController0.ddr = data.Value & 0xf0
		inp.HandController1.ddr = data.Value & 0x0f
	case "SWCHB":
	case "SWBCNT":
		inp.Panel.ddr = data.Value & 0xf0
	default:
		return true
	}

	return false
}
