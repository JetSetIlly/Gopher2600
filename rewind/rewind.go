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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia"
)

// Runner provides the rewind package the opportunity to run the emulation.
type Runner interface {
	// CatchUpLoop implementations will run the emulation until the TV returns
	// frame/scanline/horizpos values of at least the specified values.
	CatchUpLoop(frame int, scanline int, horizpos int) error
}

// State contains pointers to areas of the VCS emulation. They can be read for
// reference.
type State struct {
	level snapshotLevel

	CPU  *cpu.CPU
	Mem  *memory.Memory
	RIOT *riot.RIOT
	TIA  *tia.TIA
	TV   *television.State

	// as a consequence of how cartridge mappers have been implemented, it is
	// not possible to offer anything more than an interface to snapshotted
	// cartridge data
	cart mapper.CartSnapshot
}

// snapshotLevel indicates the level of snapshot.
type snapshotLevel int

// List of valid SnapshotLevel values.
const (
	levelReset snapshotLevel = iota
	levelFrame
	levelExecution
)

func (s State) String() string {
	if s.level == levelExecution {
		return "c"
	}
	return fmt.Sprintf("%d", s.TV.GetState(signal.ReqFramenum))
}

// the maximum number of entries to store before the earliest steps are forgotten. there
// is an overhead of two entries to facilitate appending etc.
const overhead = 2
const maxEntries = 200 + overhead

// how often a frame snapshot of the system be taken. See debugger.PushRewind()
// for why this should always be 1 for now.
const frequency = 1

// Rewind contains a history of machine states for the emulation.
type Rewind struct {
	vcs    *hardware.VCS
	runner Runner

	// circular arry of snapshotted entries
	entries [maxEntries]*State
	start   int
	end     int

	// the position of the current rewind entry and the position previous to
	// that. the previous position (prev) can be used to recrete the image that
	// would be seen at the rewind position (curr)
	curr int
	prev int

	// pointer to the comparison point
	comparison *State

	// a new frame has been triggerd. resolve as soon as possible.
	newFrame bool

	// the last call to append() was a successful ResolveNewFrame(). under
	// normal circumstances this field will be true one CPU instruction before
	// being reset.
	justAddedFrame bool
}

// NewRewind is the preferred method of initialisation for the Rewind type.
func NewRewind(vcs *hardware.VCS, runner Runner) *Rewind {
	r := &Rewind{
		vcs:    vcs,
		runner: runner,
	}
	r.vcs.TV.AddFrameTrigger(r)

	return r
}

// Reset rewind system removes all entries and takes a snapshot of the
// execution state. This should be called whenever a new cartridge is attached
// to the emulation.
func (r *Rewind) Reset() {
	r.justAddedFrame = true
	r.newFrame = false

	s := &State{
		level: levelReset,
		CPU:   r.vcs.CPU.Snapshot(),
		Mem:   r.vcs.Mem.Snapshot(),
		RIOT:  r.vcs.RIOT.Snapshot(),
		TIA:   r.vcs.TIA.Snapshot(),
		TV:    r.vcs.TV.Snapshot(),
		cart:  r.vcs.Mem.Cart.Snapshot(),
	}

	r.curr = maxEntries
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

	r.newFrame = false

	// add state only if frequency check passes
	if r.prev < len(r.entries) && r.entries[r.prev].level != levelExecution {
		if r.vcs.TV.GetState(signal.ReqFramenum)%frequency != 0 {
			return
		}
	}

	r.justAddedFrame = true

	s := &State{
		level: levelFrame,
		CPU:   r.vcs.CPU.Snapshot(),
		Mem:   r.vcs.Mem.Snapshot(),
		RIOT:  r.vcs.RIOT.Snapshot(),
		TIA:   r.vcs.TIA.Snapshot(),
		TV:    r.vcs.TV.Snapshot(),
		cart:  r.vcs.Mem.Cart.Snapshot(),
	}

	r.trim()
	r.append(s)
}

// ExecutionState takes a snapshot of the emulation's ExecutionState state. It will do
// nothing if the last call to ResolveNewFrame() resulted in a snapshot being
// taken.
func (r *Rewind) ExecutionState() {
	if r.justAddedFrame {
		return
	}

	s := &State{
		level: levelExecution,
		CPU:   r.vcs.CPU.Snapshot(),
		Mem:   r.vcs.Mem.Snapshot(),
		RIOT:  r.vcs.RIOT.Snapshot(),
		TIA:   r.vcs.TIA.Snapshot(),
		TV:    r.vcs.TV.Snapshot(),
		cart:  r.vcs.Mem.Cart.Snapshot(),
	}

	r.trim()
	r.append(s)
}

func (r *Rewind) append(s *State) {
	// append at current position
	e := r.curr + 1
	if e >= maxEntries {
		e = 0
	}

	// update entry
	r.entries[e] = s

	// note the previous position
	r.prev = r.curr

	// new position is the update point
	r.curr = e

	// next update point is recent update point plus one
	r.end = r.curr + 1
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

// chop off the most end entry if it is levelExecution.
func (r *Rewind) trim() {
	if r.entries[r.curr].level == levelExecution {
		r.end = r.curr
		if r.curr == 0 {
			r.curr = maxEntries - 1
		} else {
			r.curr--
		}
	}
}

// Frames of the current state of the rewind system.
type Frames struct {
	Start   int
	End     int
	Current int
}

// GetFrames returns the number number of snapshotted entries in the rewind system
// and the current state being pointed to (the state that is currently plumbed
// into the emulation).
func (r Rewind) GetFrames() Frames {
	e := r.end - 1
	if e < 0 {
		e += maxEntries
	}

	return Frames{
		Start:   r.entries[r.start].TV.GetState(signal.ReqFramenum),
		End:     r.entries[e].TV.GetState(signal.ReqFramenum),
		Current: r.vcs.TV.GetState(signal.ReqFramenum),
	}
}

func (r *Rewind) plumb(idx int, frame int) error {
	r.curr = idx

	// plumb will run the emulation to the specified frame, breaking on the
	// first scanline/horizpos it encounters.
	//
	// if the target is the "execution" frame then we'll update these values
	// later.
	//
	// note that bx is not zero but negative HorizClksHBlank. this is because
	// television.GetState(ReqHorizPos) returns values counting from that value
	// and not zero, as you might expect.
	bx := -specification.HorizClksHBlank
	by := 0

	// use a more specific breakpoint if entry is an "execution" entry
	if r.entries[idx].level == levelExecution {
		by = r.entries[idx].TV.GetState(signal.ReqScanline)
		bx = r.entries[idx].TV.GetState(signal.ReqHorizPos)
		idx--
		if idx < 0 {
			idx += maxEntries
		}
	}

	// if this isn't a snapshot of freshly reset machine then move position to
	// the previous state (to the one we want). after plumbing, we'll allow the
	// emulation to run to the breakpoint (specified above) of the state we do
	// want.
	if r.entries[idx].level != levelReset {
		idx--
		if idx < 0 {
			idx += maxEntries
		}
	}

	// plumb in snapshots of stored states.
	s := r.entries[idx]

	// take another snapshot of the state before plumbing. we don't want the
	// machine to change what we have stored in our state array (we learned
	// that lesson the hard way :-)
	r.vcs.CPU = s.CPU.Snapshot()
	r.vcs.Mem = s.Mem.Snapshot()
	r.vcs.RIOT = s.RIOT.Snapshot()
	r.vcs.TIA = s.TIA.Snapshot()

	r.vcs.CPU.Plumb(r.vcs.Mem)
	r.vcs.RIOT.Plumb(r.vcs.Mem.RIOT, r.vcs.Mem.TIA)
	r.vcs.TIA.Plumb(r.vcs.Mem.TIA, r.vcs.RIOT.Ports)
	r.vcs.Mem.Cart.Plumb(s.cart.Snapshot())
	r.vcs.TV.Plumb(s.TV.Snapshot())

	// make sure newFrame flag is false
	r.newFrame = false

	// not running emulation in the event of a reset but we still want an
	// updated TV image because the machine is in the freshly reset state
	if r.entries[idx].level == levelReset {
		r.vcs.TV.Reset()
		return nil
	}

	// run emulation until we reach the breakpoint of the snapshot we want
	err := r.runner.CatchUpLoop(frame, by, bx)
	if err != nil {
		return curated.Errorf("rewind", err)
	}
	err = r.vcs.TV.ForceDraw()
	if err != nil {
		return curated.Errorf("rewind", err)
	}

	return nil
}

// GotoLast sets the position to the last in the timeline.
func (r *Rewind) GotoLast() error {
	idx := r.end - 1
	if idx < 0 {
		idx += maxEntries
	}
	fn := r.entries[idx].TV.GetState(signal.ReqFramenum)
	return r.plumb(idx, fn)
}

// GotoFrame searches the timeline for the frame number. If the precise frame
// number can not be found the nearest frame will be plumbed in.
func (r *Rewind) GotoFrame(frame int) (int, error) {
	// initialise binary search
	s := r.start
	e := r.end - 1
	if e < 0 {
		e += maxEntries
	}

	// check whether request is out of bounds. plumb in nearest entry (using
	// the stored frame number rather than the requested frame number because
	// we don't want the plumb() function to run the emulation to try to catch
	// up to the requested frame)
	fn := r.entries[r.start].TV.GetState(signal.ReqFramenum)
	if frame <= fn {
		return fn, r.plumb(r.start, fn)
	}
	fn = r.entries[e].TV.GetState(signal.ReqFramenum)
	if frame >= fn {
		return fn, r.plumb(e, fn)
	}

	// because r.entries is a cirular array, there's an additional step to the
	// binary search. if start (lower) is greater then end (upper) then check
	// which half of the circular array to concentrate on.
	if r.start > e {
		fn := r.entries[maxEntries-1].TV.GetState(signal.ReqFramenum)
		if frame <= fn {
			e = maxEntries - 1
		} else {
			e = r.start - 1
			s = 0
		}
	}

	// normal binary search
	for s <= e {
		m := (s + e) / 2

		fn := r.entries[m].TV.GetState(signal.ReqFramenum)

		// check for match taking into consideration the gaps introduced by the
		// frequency value
		if frame >= fn && frame <= fn+frequency-1 {
			return fn, r.plumb(m, frame)
		}

		if frame < fn {
			e = m - 1
		}
		if frame > fn {
			s = m + 1
		}
	}

	// no change
	return r.vcs.TV.GetState(signal.ReqFramenum), nil
}

// GotoFrameCoords of current frame.
func (r *Rewind) GotoFrameCoords(scanline int, horizpos int) error {
	idx := r.curr

	// frame to which to run the catch-up loop
	frame := r.entries[idx].TV.GetState(signal.ReqFramenum)

	// start catch-up loop from previous frame
	idx = r.curr - 1
	if idx < 0 {
		idx += maxEntries
		if r.entries[idx] == nil {
			idx = r.curr
		}
	}

	// plumb in snapshots of stored states
	s := r.entries[idx]
	r.vcs.CPU = s.CPU.Snapshot()
	r.vcs.Mem = s.Mem.Snapshot()
	r.vcs.RIOT = s.RIOT.Snapshot()
	r.vcs.TIA = s.TIA.Snapshot()
	r.vcs.CPU.Plumb(r.vcs.Mem)
	r.vcs.RIOT.Plumb(r.vcs.Mem.RIOT, r.vcs.Mem.TIA)
	r.vcs.TIA.Plumb(r.vcs.Mem.TIA, r.vcs.RIOT.Ports)
	r.vcs.Mem.Cart.Plumb(s.cart.Snapshot())
	r.vcs.TV.Plumb(s.TV.Snapshot())

	// run emulation until we reach the breakpoint
	err := r.runner.CatchUpLoop(frame, scanline, horizpos)
	if err != nil {
		return curated.Errorf("rewind", err)
	}
	err = r.vcs.TV.ForceDraw()
	if err != nil {
		return curated.Errorf("rewind", err)
	}

	return nil
}

// SetComparison points comparison to the most recent rewound entry.
func (r *Rewind) SetComparison() {
	r.comparison = r.entries[r.curr]
}

// GetComparison gets a reference to current comparison point.
func (r *Rewind) GetComparison() *State {
	return r.comparison
}

// NewFrame is in an implementation of television.FrameTrigger.
func (r *Rewind) NewFrame(frameNum int, isStable bool) error {
	r.newFrame = true
	return nil
}
