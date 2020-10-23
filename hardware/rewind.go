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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/tia"
)

type snapshot struct {
	cpu  *cpu.CPU
	mem  *memory.Memory
	riot *riot.RIOT
	tia  *tia.TIA
	tv   television.TelevisionSnapshot
	cart mapper.CartSnapshot

	// is the snapshot a result of a frame snapshot request. See NewFrame()
	// function
	isCurrent bool
}

type rewind struct {
	vcs      *VCS
	steps    []snapshot
	position int

	// a new frame has been triggerd. resolve as soon as possible.
	newFrame bool

	// the last call to append() was a successful ResolveNewFrame(). under
	// normal circumstances this field will be true one CPU instruction before
	// being reset.
	justAddedFrame bool
}

const maxSteps = 300

func newRewind(vcs *VCS) *rewind {
	r := &rewind{
		vcs:   vcs,
		steps: make([]snapshot, 0, maxSteps),
	}
	r.vcs.TV.AddFrameTrigger(r)

	return r
}

// Reset rewind system to zero, taking a snapshot of the current state.
func (r *rewind) Reset() {
	r.steps = r.steps[:0]
	r.append(snapshot{
		cpu:       r.vcs.CPU.Snapshot(),
		mem:       r.vcs.Mem.Snapshot(),
		riot:      r.vcs.RIOT.Snapshot(),
		tia:       r.vcs.TIA.Snapshot(),
		tv:        r.vcs.TV.Snapshot(),
		cart:      r.vcs.Mem.Cart.Snapshot(),
		isCurrent: false,
	})
	r.justAddedFrame = true
	r.newFrame = false
}

// ResolveNewFrame is called after every CPU instruction to check whether
// a new frame has been triggered since the last call.
func (r *rewind) ResolveNewFrame() {
	if !r.newFrame {
		r.justAddedFrame = false
		return
	}

	r.justAddedFrame = true
	r.newFrame = false

	r.append(snapshot{
		cpu:       r.vcs.CPU.Snapshot(),
		mem:       r.vcs.Mem.Snapshot(),
		riot:      r.vcs.RIOT.Snapshot(),
		tia:       r.vcs.TIA.Snapshot(),
		tv:        r.vcs.TV.Snapshot(),
		cart:      r.vcs.Mem.Cart.Snapshot(),
		isCurrent: false,
	})
}

func (r *rewind) CurrentState() {
	if r.justAddedFrame {
		return
	}

	r.append(snapshot{
		cpu:       r.vcs.CPU.Snapshot(),
		mem:       r.vcs.Mem.Snapshot(),
		riot:      r.vcs.RIOT.Snapshot(),
		tia:       r.vcs.TIA.Snapshot(),
		tv:        r.vcs.TV.Snapshot(),
		cart:      r.vcs.Mem.Cart.Snapshot(),
		isCurrent: true,
	})
}

func (r *rewind) append(s snapshot) {
	if r.position >= maxSteps {
		r.steps = append(r.steps[1:], s)
	} else if len(r.steps) == 0 {
		r.steps = append(r.steps, s)
	} else {
		r.steps = append(r.steps[:r.position], s)
	}
	r.position = len(r.steps)
}

// TrimCurrent the snapshot at the current position if it is not a snapshot at a frame boundary.
func (r *rewind) TrimCurrent() {
	if len(r.steps) == 0 {
		return
	}

	if r.steps[r.position-1].isCurrent {
		r.steps = r.steps[:r.position-1]
		r.position--
	}
}

// Returns current state of the rewind. First return value is total number of
// states and the second value is the current position.
func (r rewind) State() (int, int) {
	return len(r.steps), r.position - 1
}

// Move timeline to to specified position.
func (r *rewind) SetPosition(pos int) {
	if pos >= len(r.steps) {
		pos = len(r.steps) - 1
	}

	s := r.steps[pos]
	r.vcs.CPU = s.cpu
	r.vcs.Mem = s.mem
	r.vcs.RIOT = s.riot
	r.vcs.TIA = s.tia
	r.vcs.CPU.Plumb(r.vcs.Mem)
	r.vcs.RIOT.Plumb(r.vcs.Mem.RIOT, r.vcs.Mem.TIA)
	r.vcs.TIA.Plumb(r.vcs.Mem.TIA, r.vcs.RIOT.Ports)
	r.vcs.TV.Plumb(s.tv)
	r.vcs.Mem.Cart.Plumb(s.cart)

	r.position = pos + 1
}

// NewFrame is in an implementation of television.FrameTrigger.
func (r *rewind) NewFrame(frameNum int, isStable bool) error {
	r.newFrame = true
	return nil
}
