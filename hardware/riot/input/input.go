package input

import (
	"fmt"
	"gopher2600/hardware/memory/bus"
)

// Input implements the input/output part of the RIOT (the IO in RIOT)
type Input struct {
	mem    bus.InputDeviceBus
	tiaMem bus.InputDeviceBus

	Panel   *Panel
	Player0 *Player
	Player1 *Player
}

// NewInput is the preferred method of initialisation of the Input type. Note
// that input devices require access to TIA memory as well as RIOT memory,
// breaking the abstraction somewhat, but it can't be helped. The NewInput()
// function therefore requires two arguments one to the RIOT chip bus and one
// to the TIA chip bus.
func NewInput(mem bus.ChipBus, tiaMem bus.ChipBus) (*Input, error) {
	inp := &Input{
		// we require the InputDeviceBus to memory. the following silently
		// converts to the correct bus
		// !!TODO: protect this conversion and produce a PanicError if
		// something is wrong
		mem:    mem.(bus.InputDeviceBus),
		tiaMem: tiaMem.(bus.InputDeviceBus),
	}

	inp.Panel = NewPanel(inp)
	if inp.Panel == nil {
		return nil, fmt.Errorf("can't create control panel")
	}

	inp.Player0 = NewPlayer0(inp)
	if inp.Player0 == nil {
		return nil, fmt.Errorf("can't create player 0 port")
	}

	inp.Player1 = NewPlayer1(inp)
	if inp.Player1 == nil {
		return nil, fmt.Errorf("can't create player 1 port")
	}

	return inp, nil
}

// ServiceMemory checks to see if ChipData applies to the Input type and
// updates the internal controller/panel states accordingly. Returns true if
// the ChipData was *not* serviced.
func (inp *Input) ServiceMemory(data bus.ChipData) bool {
	switch data.Name {
	case "SWCHA":
	case "SWACNT":
		inp.Player0.ddr = data.Value & 0xf0
		inp.Player1.ddr = data.Value & 0x0f
	case "SWCHB":
	case "SWBCNT":
		inp.Panel.ddr = data.Value & 0xf0
	default:
		return true
	}

	return false
}
