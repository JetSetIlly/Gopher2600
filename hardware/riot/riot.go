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

	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
)

// RIOT represents the PIA 6532 found in the VCS.
type RIOT struct {
	instance *instance.Instance

	mem chipbus.Memory

	Timer *timer.Timer
	Ports *ports.Ports
}

// NewRIOT is the preferred method of initialisation for the RIOT type.
func NewRIOT(instance *instance.Instance, mem chipbus.Memory, tiaMem chipbus.Memory) *RIOT {
	return &RIOT{
		instance: instance,
		mem:      mem,
		Timer:    timer.NewTimer(instance, mem),
		Ports:    ports.NewPorts(instance, mem, tiaMem),
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
func (riot *RIOT) Plumb(mem chipbus.Memory, tiaMem chipbus.Memory) {
	riot.mem = mem
	riot.Timer.Plumb(mem)
	riot.Ports.Plumb(mem, tiaMem)
}

func (riot *RIOT) String() string {
	s := strings.Builder{}
	s.WriteString(riot.Timer.String())
	return s.String()
}

// Step moves the state of the RIOT forward one CPU cycle.
func (riot *RIOT) Step() {
	ok, data := riot.mem.ChipHasChanged()
	if ok {
		ok = riot.Timer.Update(data)
		if ok {
			_ = riot.Ports.Update(data)
		}
	}

	riot.Timer.Step()

	// there is potentially some performance saving by calling Ports.Step()
	// less frequently. however, we must be careful because some peripherals
	// will be sensitive to this. the savekey for example is set up to be
	// updated every cycle and the paddle discharge would have to be altered.
	//
	// !!TODO: conditional calling of Ports.Step()
	riot.Ports.Step()
}

// Step moves the state of the RIOT forward one CPU cycle. Does not check to
// see if the state of RIOT memory has changed.
func (riot *RIOT) QuickStep() {
	riot.Timer.Step()

	// see comment above about riot.Ports.Step()
	riot.Ports.Step()
}
