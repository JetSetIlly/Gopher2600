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

package hardware

import (
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/tia"
)

// State stores the VCS sub-systems. It is produced by the Snapshot() function
// and can be restored with the Plumb() function
//
// Note in particular that the TV is not part of the snapshot process
type State struct {
	CPU  *cpu.CPU
	Mem  *memory.Memory
	RIOT *riot.RIOT
	TIA  *tia.TIA
}

// Snapshot creates a copy of a previously snapshotted VCS State
func (s *State) Snapshot() *State {
	return &State{
		CPU:  s.CPU.Snapshot(),
		Mem:  s.Mem.Snapshot(),
		RIOT: s.RIOT.Snapshot(),
		TIA:  s.TIA.Snapshot(),
	}
}

// Snapshot the state of the VCS sub-systems
func (vcs *VCS) Snapshot() *State {
	return &State{
		CPU:  vcs.CPU.Snapshot(),
		Mem:  vcs.Mem.Snapshot(),
		RIOT: vcs.RIOT.Snapshot(),
		TIA:  vcs.TIA.Snapshot(),
	}
}

// Plumb a previously snapshotted system
//
// The fromDifferentEmulation indicates that the State has been created by a
// different VCS emulation than the one being plumbed into
func (vcs *VCS) Plumb(state *State, fromDifferentEmulation bool) {
	if state == nil {
		panic("vcs: cannot plumb in a nil state")
	}

	// take another snapshot of the state before plumbing. we don't want the
	// machine to change what we have stored in our state array (we learned
	// that lesson the hard way :-)
	vcs.CPU = state.CPU.Snapshot()
	vcs.Mem = state.Mem.Snapshot()
	vcs.RIOT = state.RIOT.Snapshot()
	vcs.TIA = state.TIA.Snapshot()

	vcs.CPU.Plumb(vcs.Mem)
	vcs.Mem.Plumb(vcs.Env, fromDifferentEmulation)
	vcs.RIOT.Plumb(vcs.Env, vcs.Mem.RIOT, vcs.Mem.TIA)
	vcs.TIA.Plumb(vcs.Env, vcs.TV, vcs.Mem.TIA, vcs.RIOT.Ports, vcs.CPU)

	// reset peripherals after new state has been plumbed. without this,
	// controllers can feel odd if the newly plumbed state has left RIOT memory
	// in a latched state
	vcs.RIOT.Ports.ResetPeripherals()

	vcs.Input.Plumb(vcs.TV, vcs.RIOT.Ports)
}
