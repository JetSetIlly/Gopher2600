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
	"gopher2600/hardware/memory/addresses"
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

// VBlankBits represents the bits in the VBLANK register that control
// paddle-grounding and joystick-latching.  The VBLANK address is handled by
// the TIA but these bits in that register are needed by the input system.
//
// VBlankBits is instantiated by NewInput() and then a reference given
// to the TIA (by NewVCS() in the hardware package)
type VBlankBits struct {
	groundPaddles   bool
	latchFireButton bool

	// reference to the parent Input type
	inp *Input
}

// SetGroundPaddles sets the state of the groundPaddles value
func (c *VBlankBits) SetGroundPaddles(v bool) {
	c.groundPaddles = v
	c.inp.HandController0.ground()
	c.inp.HandController1.ground()
}

// SetLatchFireButton sets the state of the latchFireButton value
func (c *VBlankBits) SetLatchFireButton(v bool) {
	c.latchFireButton = v
	if !v {
		c.inp.HandController0.unlatch()
		c.inp.HandController1.unlatch()
	}
}

// Input implements the input/output part of the RIOT (the IO in RIOT)
type Input struct {
	mem        inputMemory
	VBlankBits VBlankBits

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

	// give reference to new Input type to its VBlankBits
	inp.VBlankBits.inp = inp

	inp.Panel = NewPanel(&inp.mem)
	if inp.Panel == nil {
		return nil, fmt.Errorf("can't create control panel")
	}

	inp.HandController0 = NewHandController0(&inp.mem, &inp.VBlankBits)
	if inp.HandController0 == nil {
		return nil, fmt.Errorf("can't create player 0 port")
	}

	inp.HandController1 = NewHandController1(&inp.mem, &inp.VBlankBits)
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
		inp.HandController0.readKeypad(data.Value)
		inp.HandController1.readKeypad(data.Value)

		// write data back to memory
		inp.mem.riot.InputDeviceWrite(addresses.SWCHA, data.Value, 0x00)

	case "SWACNT":
		inp.HandController0.setDDR(data.Value)
		inp.HandController1.setDDR(data.Value)

		// write data back to memory
		inp.mem.riot.InputDeviceWrite(addresses.SWACNT, data.Value, 0x00)

		// update SWCHA too !!TODO: not sure if this is correct just yet
		inp.mem.riot.InputDeviceWrite(addresses.SWCHA, 0xff, data.Value)

	case "SWCHB":
		panic("Port B; console switches (hardwired as input)")
	case "SWBCNT":
		panic("Port B DDR (hardwired as input)")
	default:
		return true
	}

	return false
}

// Step input state forward one cycle
func (inp *Input) Step() {
	// not much to do here because most input operations happen on demand.
	// recharging of the paddle capacitors however happens (a little bit) every
	// step.
	inp.HandController0.recharge()
	inp.HandController1.recharge()
}
