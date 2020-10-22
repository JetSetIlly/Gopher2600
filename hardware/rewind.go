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

	// is the snapshot a result of a frame snapshot request. See NewFrame()
	// function
	frame bool
}

type rewind struct {
	vcs      *VCS
	steps    []snapshot
	position int

	newFrame               bool
	lastAppendFromNewFrame bool
}

const maxSteps = 300

func newRewind(vcs *VCS) *rewind {
	r := &rewind{
		vcs:   vcs,
		steps: make([]snapshot, 0, maxSteps),
	}
	r.vcs.TV.AddFrameTrigger(r)
	r.newFrame = true
	return r
}

// Append should only be called on a CPU instruction boundary. If we call it
// every CPU instruction then we can control when we save almost entirely
// within this function.
//
// Currently, the policy is to create a snapshot every frame. The NewFrame()
// function (implements television.PixelRenderer) sets the newFrame flag
// which is checked on the next instruction boundary. We do this because
// NewFrame() can be called mid-instruction.
//
// The force flag appends a snapshot regardless of the newFrame flag.
func (r *rewind) Append(force bool) {
	if !force && !r.newFrame {
		r.lastAppendFromNewFrame = false
		return
	}

	if force && r.lastAppendFromNewFrame {
		return
	}

	s := snapshot{
		CPU:   r.vcs.CPU.Snapshot(),
		Mem:   r.vcs.Mem.Snapshot(),
		TIA:   r.vcs.TIA.Snapshot(),
		RIOT:  r.vcs.RIOT.Snapshot(),
		TV:    r.vcs.TV.Snapshot(),
		frame: r.newFrame,
	}

	r.lastAppendFromNewFrame = r.newFrame
	r.newFrame = false

	if r.position >= maxSteps {
		r.steps = append(r.steps[1:], s)
	} else if len(r.steps) == 0 {
		r.steps = append(r.steps, s)
	} else {
		r.steps = append(r.steps[:r.position], s)
	}
	r.position = len(r.steps)
}

// Trim the snapshot at the current position if it is not a snapshot at a frame boundary.
func (r *rewind) TrimNonFrame() {
	if len(r.steps) == 0 {
		return
	}

	if !r.steps[r.position-1].frame {
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
	r.vcs.CPU = s.CPU
	r.vcs.Mem = s.Mem
	r.vcs.TIA = s.TIA
	r.vcs.RIOT = s.RIOT

	r.vcs.TV.RestoreSnapshot(s.TV)
	r.vcs.CPU.Plumb(r.vcs.Mem)
	r.vcs.TIA.Plumb(r.vcs.Mem.TIA)
	r.vcs.RIOT.Plumb(r.vcs.Mem.RIOT, r.vcs.Mem.TIA)

	r.position = pos + 1
}

// NewFrame is in an implementation of television.FrameTrigger.
func (r *rewind) NewFrame(frameNum int, isStable bool) error {
	r.newFrame = true
	return nil
}
