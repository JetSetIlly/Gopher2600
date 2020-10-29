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

package rewind

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/reflection"
)

// Snapshot contains pointers to areas of the VCS emulation. They can be read
// for reference.
type Snapshot struct {
	CPU  *cpu.CPU
	Mem  *memory.Memory
	RIOT *riot.RIOT
	TIA  *tia.TIA
	TV   *television.State

	// must be checked for nil
	Reflection *reflection.State

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
	return fmt.Sprintf("%d", s.TV.GetState(signal.ReqFramenum))
}

// the maximum number of entries to store before the earliest steps are forgotten. there
// is an overhead of two entries to facilitate appending etc.
const overhead = 2
const maxEntries = 20 + overhead

// Rewind contains a history of machine states for the emulation.
type Rewind struct {
	vcs *hardware.VCS
	ref *reflection.Monitor

	// list of snapshotted entries
	entries [maxEntries]*Snapshot
	start   int
	end     int
	prev    int

	// pointer to the comparison point
	comparison *Snapshot

	// a new frame has been triggerd. resolve as soon as possible.
	newFrame bool

	// the last call to append() was a successful ResolveNewFrame(). under
	// normal circumstances this field will be true one CPU instruction before
	// being reset.
	justAddedFrame bool
}

// NewRewind is the preferred method of initialisation for the Rewind type.
func NewRewind(vcs *hardware.VCS, reflector *reflection.Monitor) *Rewind {
	r := &Rewind{
		vcs: vcs,
		ref: reflector,
	}
	r.vcs.TV.AddFrameTrigger(r)

	return r
}

// Reset rewind system removes all entries and takes a snapshot of the current
// state. This should be called whenever a new cartridge is attached to the
// emulation.
func (r *Rewind) Reset() {
	r.justAddedFrame = true
	r.newFrame = false

	s := &Snapshot{
		CPU:       r.vcs.CPU.Snapshot(),
		Mem:       r.vcs.Mem.Snapshot(),
		RIOT:      r.vcs.RIOT.Snapshot(),
		TIA:       r.vcs.TIA.Snapshot(),
		TV:        r.vcs.TV.Snapshot(),
		cart:      r.vcs.Mem.Cart.Snapshot(),
		isCurrent: false,
	}

	if r.ref != nil {
		s.Reflection = r.ref.Snapshot()
	}

	r.prev = maxEntries
	r.append(s)

	// first comparison is to the snapshot of the reset machine
	r.comparison = r.entries[0]
}

// Check should be called after every CPU instruction to check whether a new
// frame has been triggered since the last call. Delaying a call to this
// function may result in sub-optimal results.
func (r *Rewind) Check() {
	if !r.newFrame {
		r.justAddedFrame = false
		return
	}

	r.justAddedFrame = true
	r.newFrame = false

	s := &Snapshot{
		CPU:       r.vcs.CPU.Snapshot(),
		Mem:       r.vcs.Mem.Snapshot(),
		RIOT:      r.vcs.RIOT.Snapshot(),
		TIA:       r.vcs.TIA.Snapshot(),
		TV:        r.vcs.TV.Snapshot(),
		cart:      r.vcs.Mem.Cart.Snapshot(),
		isCurrent: false,
	}

	if r.ref != nil {
		s.Reflection = r.ref.Snapshot()
	}

	r.trim()
	r.append(s)
}

// CurrentState takes a snapshot of the emulation's current state. It will do
// nothing if the last call to ResolveNewFrame() resulted in a snapshot being
// taken.
func (r *Rewind) CurrentState() {
	if r.justAddedFrame {
		return
	}

	s := &Snapshot{
		CPU:       r.vcs.CPU.Snapshot(),
		Mem:       r.vcs.Mem.Snapshot(),
		RIOT:      r.vcs.RIOT.Snapshot(),
		TIA:       r.vcs.TIA.Snapshot(),
		TV:        r.vcs.TV.Snapshot(),
		cart:      r.vcs.Mem.Cart.Snapshot(),
		isCurrent: true,
	}

	if r.ref != nil {
		s.Reflection = r.ref.Snapshot()
	}

	r.trim()
	r.append(s)
}

func (r *Rewind) append(s *Snapshot) {
	// append at previous point plus one
	e := r.prev + 1
	if e >= maxEntries {
		e -= maxEntries
	}

	// update entry
	r.entries[e] = s

	// new previous is the update point
	r.prev = e

	// next update point is recent update point plus one
	r.end = r.prev + 1
	if r.end >= maxEntries {
		r.end = 0
	}

	// push start index along
	if r.end == r.start {
		r.start++
		if r.start >= maxEntries {
			r.start = 0
		}
	}
}

// chop off the most recent entry if the isCurrent flag is set.
func (r *Rewind) trim() {
	if r.entries[r.prev].isCurrent {
		r.end = r.prev
		if r.prev == 0 {
			r.prev = maxEntries - 1
		} else {
			r.prev--
		}
	}
}

// State returns the number number of snapshotted entries in the rewind system
// and the current state being pointed to (the state that is currently plumbed
// into the emulation).
func (r Rewind) State() (int, int) {
	// number of entries is always equal to end point minus start point,
	// adjusted for negative numbers.
	n := r.end - r.start - 1
	if n < 0 {
		n += maxEntries
	}

	i := r.prev - r.start - 1
	if i < 0 {
		i += maxEntries
	}

	return n, i
}

// SetPosition sets the rewind system to the specified position. That state
// will be plumbed into the emulation.
func (r *Rewind) SetPosition(pos int) {
	pos += r.start + 1
	if pos >= maxEntries {
		pos -= maxEntries
	}
	r.plumb(pos)
	r.prev = pos
}

func (r Rewind) plumb(idx int) {
	s := r.entries[idx]

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

	if r.ref != nil && s.Reflection != nil {
		r.ref.Plumb(s.Reflection.Snapshot())
	}
}

// GotoCurrent sets the position to the last in the timeline.
func (r *Rewind) GotoCurrent() {
	r.prev = r.end - 1
	if r.prev < 0 {
		r.prev += maxEntries
	}
	r.plumb(r.prev)
}

// GotoFrame searches the timeline for the frame number. Goes to nearest frame
// if frame number is not present. Returns true if exact frame number was found
// and false if not.
func (r *Rewind) GotoFrame(frame int) bool {
	exactMatch := false
	p := r.start

	for i := 0; i < maxEntries; i++ {
		if r.entries[i] != nil {
			fn := r.entries[i].TV.GetState(signal.ReqFramenum)

			if frame == fn {
				p = i
				exactMatch = true
				break // for loop
			}

			if frame > fn {
				p = i
			}
		}
	}

	r.prev = p
	r.plumb(r.prev)

	return exactMatch
}

// SetComparison points comparison to the most recent rewound entry.
func (r *Rewind) SetComparison() {
	r.comparison = r.entries[r.prev]
}

// GetComparison gets a reference to current comparison point.
func (r *Rewind) GetComparison() *Snapshot {
	return r.comparison
}

// NewFrame is in an implementation of television.FrameTrigger.
func (r *Rewind) NewFrame(frameNum int, isStable bool) error {
	r.newFrame = true
	return nil
}
