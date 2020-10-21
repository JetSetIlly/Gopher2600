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

package riot

import (
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
)

// RIOT represents the PIA 6532 found in the VCS.
type RIOT struct {
	mem bus.ChipBus

	Timer *timer.Timer
	Ports *ports.Ports
}

// NewRIOT is the preferred method of initialisation for the RIOT type.
func NewRIOT(mem bus.ChipBus, tiaMem bus.ChipBus) *RIOT {
	return &RIOT{
		mem:   mem,
		Timer: timer.NewTimer(mem),
		Ports: ports.NewPorts(mem, tiaMem),
	}
}

// Snapshot creates a copy of the RIOT in its current state.
func (riot *RIOT) Snapshot() *RIOT {
	n := *riot
	n.Timer = riot.Timer.Snapshot()
	n.Ports = riot.Ports.Snapshot()
	return &n
}

// Plumb new ChipBusses into the RIOT.
func (riot *RIOT) Plumb(mem bus.ChipBus, tiaMem bus.ChipBus) {
	riot.mem = mem
	riot.Timer.Plumb(mem)
	riot.Ports.Plumb(mem, tiaMem)
}

func (riot RIOT) String() string {
	s := strings.Builder{}
	s.WriteString(riot.Timer.String())
	return s.String()
}

// UpdateRIOT checks for the most recent write by the CPU to the RIOT memory
// registers.
func (riot *RIOT) UpdateRIOT() {
	ok, data := riot.mem.ChipRead()
	if !ok {
		return
	}

	ok = riot.Timer.Update(data)
	if !ok {
		return
	}

	_ = riot.Ports.Update(data)
}

// Step moves the state of the RIOT forward one video cycle.
func (riot *RIOT) Step() {
	riot.UpdateRIOT()
	riot.Timer.Step()
	riot.Ports.Step()
}
