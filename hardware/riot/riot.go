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

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
	"github.com/jetsetilly/gopher2600/logger"
)

// RIOT represents the PIA 6532 found in the VCS.
type RIOT struct {
	env *environment.Environment

	mem chipbus.Memory

	Timer *timer.Timer
	Ports *ports.Ports
}

// NewRIOT is the preferred method of initialisation for the RIOT type.
func NewRIOT(env *environment.Environment, mem chipbus.Memory, tiaMem chipbus.Memory) *RIOT {
	return &RIOT{
		env:   env,
		mem:   mem,
		Timer: timer.NewTimer(env, mem),
		Ports: ports.NewPorts(env, mem, tiaMem),
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
func (riot *RIOT) Step(reg chipbus.ChangedRegister) {
	update := riot.Timer.Update(reg)
	if update {
		update = riot.Ports.Update(reg)
		if update {
			logger.Logf("riot", "memory altered to no affect (%04x=%02x)", reg.Address, reg.Value)
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
