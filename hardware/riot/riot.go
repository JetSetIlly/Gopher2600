package riot

import (
	"gopher2600/hardware/memory/bus"
	"gopher2600/hardware/riot/input"
	"gopher2600/hardware/riot/timer"
	"strings"
)

// RIOT represents the PIA 6532 found in the VCS
type RIOT struct {
	mem bus.ChipBus

	Timer *timer.Timer
	Input *input.Input
}

// NewRIOT is the preferred method of initialisation for the RIOT type
func NewRIOT(mem bus.ChipBus, tiaMem bus.ChipBus) (*RIOT, error) {
	var err error

	riot := &RIOT{mem: mem}
	riot.Timer = timer.NewTimer(mem)
	riot.Input, err = input.NewInput(mem, tiaMem)
	if err != nil {
		return nil, err
	}

	return riot, nil
}

func (riot RIOT) String() string {
	s := strings.Builder{}
	s.WriteString(riot.Timer.String())
	return s.String()
}

// ReadMemory checks for the most recent write by the CPU to the RIOT memory
// registers
func (riot *RIOT) ReadMemory() {
	serviceMemory, data := riot.mem.ChipRead()
	if !serviceMemory {
		return
	}

	serviceMemory = riot.Timer.ServiceMemory(data)
	if !serviceMemory {
		return
	}

	_ = riot.Input.ServiceMemory(data)
}

// Step moves the state of the RIOT forward one video cycle
func (riot *RIOT) Step() {
	riot.ReadMemory()
	riot.Timer.Step()
}
