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
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/tia"
)

type snapshot struct {
	CPU  *cpu.CPU
	Mem  *memory.Memory
	TIA  *tia.TIA
	RIOT *riot.RIOT
	TV   television.TelevisionState
}

type rewind struct {
	vcs      *VCS
	steps    []snapshot
	position int

	appendNewFrame bool
}

const maxSteps = 300

func newRewind(vcs *VCS) *rewind {
	r := &rewind{
		vcs:   vcs,
		steps: make([]snapshot, 0, maxSteps),
	}
	r.vcs.TV.AddFrameTrigger(r)
	r.appendNewFrame = true
	return r
}

// Append should only be called on a CPU instruction boundary. If we call it
// every CPU instruction then we can control when we save almost entirely
// within this function.
//
// Currently, the policy is to create a snapshot every frame. The NewFrame()
// function (implements television.PixelRenderer) sets the appendNewFrame flag
// which is checked on the next instruction boundary. We do this because
// NewFrame() can be called mid-instruction.
func (r *rewind) Append() {
	if !r.appendNewFrame {
		return
	}
	r.appendNewFrame = false

	s := snapshot{
		CPU:  r.vcs.CPU.Snapshot(),
		Mem:  r.vcs.Mem.Snapshot(),
		TIA:  r.vcs.TIA.Snapshot(),
		RIOT: r.vcs.RIOT.Snapshot(),
		TV:   r.vcs.TV.Snapshot(),
	}

	if r.position >= maxSteps {
		r.steps = append(r.steps[1:], s)
	} else {
		r.steps = append(r.steps[:r.position], s)
	}
	r.position = len(r.steps)
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
	r.position = pos

	s := r.steps[r.position]
	r.vcs.CPU = s.CPU
	r.vcs.Mem = s.Mem
	r.vcs.TIA = s.TIA
	r.vcs.RIOT = s.RIOT

	r.vcs.TV.RestoreSnapshot(s.TV)
	r.vcs.CPU.Plumb(r.vcs.Mem)
	r.vcs.TIA.Plumb(r.vcs.Mem.TIA)
	r.vcs.RIOT.Plumb(r.vcs.Mem.RIOT, r.vcs.Mem.TIA)
}

// NewFrame is in an implementation of television.FrameTrigger.
func (r *rewind) NewFrame(frameNum int, isStable bool) error {
	r.appendNewFrame = true
	return nil
}
