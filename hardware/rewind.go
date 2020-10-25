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
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/tia"
)

// Snapshot contains pointers to areas of the VCS emulation. They can be read
// for reference.
type Snapshot struct {
	CPU  *cpu.CPU
	Mem  *memory.Memory
	RIOT *riot.RIOT
	TIA  *tia.TIA
	TV   *television.State

	// as a consequence of how cartridge mappers have been implemented, it is
	// not possible to offer anything more than an interface to snapshotted
	// cartridge data
	cart mapper.CartSnapshot

	// is the snapshot a result of a frame snapshot request. See NewFrame()
	// function
	isCurrent bool
}

func (s Snapshot) String() string {
	if s.isCurrent {
		return "c"
	}
	return fmt.Sprintf("%d", s.TV.GetState(television.ReqFramenum))
}

type rewind struct {
	vcs      *VCS
	steps    []Snapshot
	position int

	// pointer to the comparison point
	comparison *Snapshot

	// a new frame has been triggerd. resolve as soon as possible.
	newFrame bool

	// the last call to append() was a successful ResolveNewFrame(). under
	// normal circumstances this field will be true one CPU instruction before
	// being reset.
	justAddedFrame bool
}

// the maximum number of steps to store before the earliest steps are
// forgotten.
const maxRewindSteps = 100

func newRewind(vcs *VCS) *rewind {
	r := &rewind{
		vcs:   vcs,
		steps: make([]Snapshot, 0, maxRewindSteps),
	}
	r.vcs.TV.AddFrameTrigger(r)

	return r
}

// Reset rewind system to zero, taking a snapshot of the current state.
func (r *rewind) Reset() {
	r.steps = r.steps[:0]
	r.append(Snapshot{
		CPU:       r.vcs.CPU.Snapshot(),
		Mem:       r.vcs.Mem.Snapshot(),
		RIOT:      r.vcs.RIOT.Snapshot(),
		TIA:       r.vcs.TIA.Snapshot(),
		TV:        r.vcs.TV.Snapshot(),
		cart:      r.vcs.Mem.Cart.Snapshot(),
		isCurrent: false,
	})
	r.justAddedFrame = true
	r.newFrame = false

	// first comparison is to the snapshot of the reset machine
	r.comparison = &r.steps[0]
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

	r.append(Snapshot{
		CPU:       r.vcs.CPU.Snapshot(),
		Mem:       r.vcs.Mem.Snapshot(),
		RIOT:      r.vcs.RIOT.Snapshot(),
		TIA:       r.vcs.TIA.Snapshot(),
		TV:        r.vcs.TV.Snapshot(),
		cart:      r.vcs.Mem.Cart.Snapshot(),
		isCurrent: false,
	})
}

func (r *rewind) CurrentState() {
	if r.justAddedFrame {
		return
	}

	r.append(Snapshot{
		CPU:       r.vcs.CPU.Snapshot(),
		Mem:       r.vcs.Mem.Snapshot(),
		RIOT:      r.vcs.RIOT.Snapshot(),
		TIA:       r.vcs.TIA.Snapshot(),
		TV:        r.vcs.TV.Snapshot(),
		cart:      r.vcs.Mem.Cart.Snapshot(),
		isCurrent: true,
	})
}

func (r *rewind) append(s Snapshot) {
	if r.position == len(r.steps) {
		r.trim()
		r.steps = append(r.steps, s)
	} else {
		r.steps = append(r.steps[:r.position], s)
	}

	// maintain maximum length
	if len(r.steps) > maxRewindSteps {
		r.steps = r.steps[1:]
	}

	r.position = len(r.steps)
}

func (r *rewind) trim() {
	if len(r.steps) < 1 {
		return
	}

	if r.steps[len(r.steps)-1].isCurrent {
		r.steps = r.steps[:len(r.steps)-1]
		r.position = len(r.steps)
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

	// plumb in snapshots of stored states. we don't want the machine to change
	// what we have stored in our state array (we learned that lesson the hard
	// way :-)
	r.vcs.CPU = s.CPU.Snapshot()
	r.vcs.Mem = s.Mem.Snapshot()
	r.vcs.RIOT = s.RIOT.Snapshot()
	r.vcs.TIA = s.TIA.Snapshot()
	r.vcs.CPU.Plumb(r.vcs.Mem)
	r.vcs.RIOT.Plumb(r.vcs.Mem.RIOT, r.vcs.Mem.TIA)
	r.vcs.TIA.Plumb(r.vcs.Mem.TIA, r.vcs.RIOT.Ports)
	r.vcs.TV.Plumb(s.TV.Snapshot())
	r.vcs.Mem.Cart.Plumb(s.cart.Snapshot())

	r.position = pos + 1
}

// GotoCurrent sets the position to the last in the timeline.
func (r *rewind) GotoCurrent() {
	r.SetPosition(len(r.steps))
}

// GotoFrame searches the timeline for the frame number. Goes to nearest frame
// if frame number is not present. Returns true if exact frame number was found
// and false if not.
func (r *rewind) GotoFrame(frame int) bool {
	// binary search for frame number
	b := 0
	t := len(r.steps) - 1
	for b <= t {
		m := (t + b) / 2

		if r.steps[m].TV.GetState(television.ReqFramenum) == frame {
			r.SetPosition(m)
			return true
		}

		if r.steps[m].TV.GetState(television.ReqFramenum) < frame {
			b = m + 1
		} else if r.steps[m].TV.GetState(television.ReqFramenum) > frame {
			t = m - 1
		}
	}

	r.SetPosition(b)
	return false
}

// SetComparison points comparison to the most recent rewound entry.
func (r *rewind) SetComparison() {
	r.comparison = &r.steps[len(r.steps)-1]
}

// GetComparison gets a reference to current comparison point.
func (r *rewind) GetComparison() *Snapshot {
	return r.comparison
}

// NewFrame is in an implementation of television.FrameTrigger.
func (r *rewind) NewFrame(frameNum int, isStable bool) error {
	r.newFrame = true
	return nil
}
