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
	"strings"

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
	"github.com/jetsetilly/gopher2600/logger"
)

// Runner provides the rewind package the opportunity to run the emulation.
type Runner interface {
	// CatchUpLoop implementations will run the emulation until continueCheck
	// returns false.
	//
	// Note that the TV's frame limiter is turned off before CatchUpLoop() is
	// called by the rewind system (and turned back to the previous setting
	// afterwards).
	//
	// CatchUpLoop may choose to service events (GUI events etc.) while it is
	// iterating but depending on required performance this may not be
	// necessary.
	CatchUpLoop(continueCheck func() bool) error
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
	// reset and boundary entries should only even appear once at the start of
	// the history, it at all.
	levelReset snapshotLevel = iota
	levelBoundary

	// there can be many frame entries in the rewind history.
	levelFrame

	// execution entries should only ever appear once at the end of the
	// history, if at all.
	levelExecution

	// adhoc entries should never appear in the history.
	levelAdhoc
)

func (s State) String() string {
	switch s.level {
	case levelReset:
		return "r"
	case levelBoundary:
		return "b"
	case levelExecution:
		return "e"
	case levelAdhoc:
		return "c"
	}
	return fmt.Sprintf("%d", s.TV.GetState(signal.ReqFramenum))
}

// an overhead of two is required. (1) to accommodate the end index required for
// effective appending; (2) we can't generate a screen for the first entry in
// the history, unless it's a reset entry, so we do not allow the rewind system
// to move to that frame.
const overhead = 2

// Rewind contains a history of machine states for the emulation.
type Rewind struct {
	vcs    *hardware.VCS
	runner Runner

	// prefs for the rewind system
	Prefs *Preferences

	// circular arry of snapshotted entries
	entries []*State
	start   int
	end     int

	// the point at which new entries will be added
	splice int

	// pointer to the comparison point
	comparison *State

	// adhoc is a special snapshot of a state that cannot be found in the
	// entries array. it is used to speed up consecutive calls to GotoCoords()
	adhoc *State

	// a new frame has been triggered. resolve as soon as possible.
	newFrame bool

	// a snapshot has just been added by the Check() function. we use this to
	// prevent another snapshot being taken by ExecutionState(). rarely comes
	// into play but it prevents what would essentially be a duplicate entry
	// being added.
	justAddedFrame bool

	// the number frames since snapshot (not counting levelExecution
	// snapshots)
	framesSinceSnapshot int

	// a rewind boundary has been detected. call restart() on next frame.
	boundaryNextFrame bool
}

// NewRewind is the preferred method of initialisation for the Rewind type.
func NewRewind(vcs *hardware.VCS, runner Runner) (*Rewind, error) {
	r := &Rewind{
		vcs:    vcs,
		runner: runner,
	}

	var err error

	r.Prefs, err = newPreferences(r)
	if err != nil {
		return nil, curated.Errorf("rewind", err)
	}

	r.vcs.TV.AddFrameTrigger(r)
	r.allocate()

	return r, nil
}

func (r *Rewind) allocate() {
	r.entries = make([]*State, r.Prefs.MaxEntries.Get().(int)+overhead)
	r.restart(levelReset)
}

func (r *Rewind) String() string {
	s := strings.Builder{}

	i := r.start
	for i < r.end && i < len(r.entries) {
		e := r.entries[i]
		if e != nil {
			s.WriteString(fmt.Sprintf("%s ", e.String()))
		}
		i++
	}

	if i != r.end {
		i = 0
		for i < r.end {
			e := r.entries[i]
			if e != nil {
				s.WriteString(fmt.Sprintf("%s ", e.String()))
			}
			i++
		}
	}

	return s.String()
}

func (r *Rewind) snapshot(level snapshotLevel) *State {
	return &State{
		level: level,
		CPU:   r.vcs.CPU.Snapshot(),
		Mem:   r.vcs.Mem.Snapshot(),
		RIOT:  r.vcs.RIOT.Snapshot(),
		TIA:   r.vcs.TIA.Snapshot(),
		TV:    r.vcs.TV.Snapshot(),
		cart:  r.vcs.Mem.Cart.Snapshot(),
	}
}

// Reset rewind system removes all entries and takes a snapshot of the
// execution state. This should be called whenever a new cartridge is attached
// to the emulation.
func (r *Rewind) Reset() {
	r.restart(levelReset)
}

func (r *Rewind) restart(level snapshotLevel) {
	// nillify all entries
	for i := range r.entries {
		r.entries[i] = nil
	}

	r.newFrame = false
	r.justAddedFrame = true
	r.framesSinceSnapshot = 0

	// this arrangement of the three history indexes means that there is no
	// special conditions in the append() function.
	//
	// start and end are equal to begin with. the first call to append() below
	// will add the new State at the current end point and then advance the end
	// index ready for the next append(). this means that the entry appended
	// will be a index start
	r.start = 1
	r.end = 1

	// the splice point is checked to see if it is an execution
	// entry and is chopped off if it is. the insertion of a sparse boundary
	// entry means we don't have to check for nil
	//
	// the append function will move the splice index to start
	r.splice = 0
	r.entries[r.splice] = &State{level: levelBoundary}

	// add current state as first entry
	r.append(r.snapshot(level))

	// first comparison is to the snapshot of the reset machine
	r.comparison = r.entries[r.start]

	// this isn't really neede but if feels good to remove the boundary entry
	// added at the initial splice index.
	r.entries[0] = nil
}

// Check should be called after every CPU instruction to check whether a new
// frame has been triggered since the last call. Delaying a call to this
// function may result in sub-optimal results.
func (r *Rewind) Check() {
	r.boundaryNextFrame = r.boundaryNextFrame || r.vcs.Mem.Cart.RewindBoundary()

	if !r.newFrame {
		r.justAddedFrame = false
		return
	}
	r.newFrame = false

	if r.boundaryNextFrame {
		r.boundaryNextFrame = false
		r.restart(levelBoundary)
		logger.Log("rewind", fmt.Sprintf("boundary added at frame %d", r.vcs.TV.GetState(signal.ReqFramenum)))
		return
	}

	// add state only if frequency check passes
	r.framesSinceSnapshot++
	if r.framesSinceSnapshot%r.Prefs.Freq.Get().(int) != 0 {
		return
	}

	r.justAddedFrame = true
	r.framesSinceSnapshot = 0

	r.append(r.snapshot(levelFrame))
}

// ExecutionState takes a snapshot of the emulation's ExecutionState state. It will do
// nothing if the last call to ResolveNewFrame() resulted in a snapshot being
// taken.
func (r *Rewind) ExecutionState() {
	if r.justAddedFrame {
		return
	}

	r.append(r.snapshot(levelExecution))
}

func (r *Rewind) append(s *State) {
	// chop off the end entry if it is in execution entry. we must do this
	// before any further appending. this is enough to ensure that there is
	// never more than one execution entry in the history.
	if r.entries[r.splice].level == levelExecution {
		r.end = r.splice
		if r.splice == 0 {
			r.splice = len(r.entries) - 1
		} else {
			r.splice--
		}
	}

	// append at current position
	e := r.splice + 1
	if e >= len(r.entries) {
		e = 0
	}

	// update entry
	r.entries[e] = s

	// new position is the update point
	r.splice = e

	// next update point is recent update point plus one
	r.end = r.splice + 1
	if r.end >= len(r.entries) {
		r.end = 0
	}

	// push start index along
	if r.end == r.start {
		r.start++
		if r.start >= len(r.entries) {
			r.start = 0
		}
	}
}

// plumb in state found at index. splice point will be updated. remaining
// arguments as in plumbState().
func (r *Rewind) plumb(idx, frame, scanline, horizpos int) error {
	// current index is the index we're plumbing in. this has nothing to do
	// with the frame number (especially important to remember if frequency is
	// greater than 1)
	r.splice = idx

	s := r.entries[idx]
	startingFrame := s.TV.GetState(signal.ReqFramenum)

	// plumb in selected entry
	err := r.plumbState(s, frame, scanline, horizpos)
	if err != nil {
		return err
	}

	// update frames since snapshot
	r.framesSinceSnapshot = r.vcs.TV.GetState(signal.ReqFramenum) - startingFrame - 1

	return nil
}

// plumb in state supplied as the argument. catch-up loop will halt as soon as
// possible after frame/scanline/horizpos is reached or surpassed
//
// note that this will not update the splice point up update the
// framesSinceSnapshot value. use plumb() with an index into the history for
// that.
func (r *Rewind) plumbState(s *State, frame, scanline, horizpos int) error {
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

	// if this is a reset entry then TV must be reset
	if s.level == levelReset {
		err := r.vcs.TV.Reset()
		if err != nil {
			return curated.Errorf("rewind", err)
		}
	}

	// turn off TV's fps frame limiter
	cap := r.vcs.TV.SetFPSCap(false)
	defer r.vcs.TV.SetFPSCap(cap)

	// snapshot adhoc frame as soon as convenient. not required when snapshot
	// frequency is one
	adhocSnapshotted := r.Prefs.Freq.Get().(int) == 1

	continueCheck := func() bool {
		nf := r.vcs.TV.GetState(signal.ReqFramenum)
		ny := r.vcs.TV.GetState(signal.ReqScanline)
		nx := r.vcs.TV.GetState(signal.ReqHorizPos)

		if !adhocSnapshotted && nf == frame-1 {
			r.adhoc = r.snapshot(levelAdhoc)
			adhocSnapshotted = true
		}

		tooFar := nf > frame || (nf == frame && ny > scanline) || (nf == frame && ny == scanline && nx >= horizpos)
		return !tooFar
	}

	// run emulation until continueCheck returns false
	err := r.runner.CatchUpLoop(continueCheck)
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
		idx += len(r.entries)
	}

	frame := r.entries[idx].TV.GetState(signal.ReqFramenum)
	horizpos := -specification.HorizClksHBlank
	scanline := 0

	// use more specific scanline/horizpos values if entry is an "execution" entry
	if r.entries[idx].level == levelExecution {
		scanline = r.entries[idx].TV.GetState(signal.ReqScanline)
		horizpos = r.entries[idx].TV.GetState(signal.ReqHorizPos)
	}

	// make adjustments to the index so we plumbing from a suitable place
	idx--
	if idx < 0 {
		idx += len(r.entries)
	}

	// boundary checks to make sure we haven't gone back past the beginning of
	// the circular array
	if r.entries[idx] == nil {
		idx = r.start
	}

	return r.plumb(idx, frame, scanline, horizpos)
}

// GotoFrame searches the timeline for the frame number. If the precise frame
// number can not be found the nearest frame will be plumbed in.
func (r *Rewind) GotoFrame(frame int) error {
	var idx int
	var last bool
	idx, frame, last = r.findFrameIndex(frame)

	// it is more appropriate to plumb with GotoLast() if last is true
	if last {
		return r.GotoLast()
	}

	// plumb in index. the frame argument to the plumb() function is
	// the frame that has been requested, not the search frame
	return r.plumb(idx, frame, 0, -specification.HorizClksHBlank)
}

// find index nearest to the requested frame. returns the index and the frame
// number that is actually possible with the rewind system.
//
// the last value indicates that the requested frame is past the end of the
// history. in those instances, the returned frame number can be used for the
// plumbing operation or because last==true the GotoLast() can be used for a
// more natural feeling result.
func (r *Rewind) findFrameIndex(frame int) (idx int, fr int, last bool) {
	// the binary search is looking for the frame before the one that has been
	// requested. this is so that we can generate the pixels that will be on
	// the screen at the beginning of the request frame.
	sf := frame
	if sf > 0 {
		sf--
	}

	// initialise binary search
	s := r.start
	e := r.end - 1
	if e < 0 {
		e += len(r.entries)
	}

	// check whether request is out of bounds of the rewind history. if it is
	// then plumb in the nearest entry
	fn := r.entries[s].TV.GetState(signal.ReqFramenum)
	if sf < fn {
		return s, fn + 1, false
	}

	fn = r.entries[e].TV.GetState(signal.ReqFramenum)
	if sf >= fn {
		e--
		if e < 0 {
			e += len(r.entries)
		}
		if r.entries[e] == nil {
			return r.start, fn, true
		}
		return e, fn, true
	}

	// because r.entries is a cirular array, there's an additional step to the
	// binary search. if start (lower) is greater then end (upper) then check
	// which half of the circular array to concentrate on.
	if r.start > e {
		fn := r.entries[len(r.entries)-1].TV.GetState(signal.ReqFramenum)
		if sf <= fn {
			e = len(r.entries) - 1
		} else {
			e = r.start - 1
			s = 0
		}
	}

	// normal binary search
	for s <= e {
		idx := (s + e) / 2

		fn := r.entries[idx].TV.GetState(signal.ReqFramenum)

		// check for match, taking into consideration the gaps introduced by
		// the frequency value
		if sf >= fn && sf <= fn+r.Prefs.Freq.Get().(int)-1 {
			return idx, frame, false
		}

		if sf < fn {
			e = idx - 1
		}
		if sf > fn {
			s = idx + 1
		}
	}

	logger.Log("rewind", "seemingly impossible failure of binary search")
	return e, frame, false
}

// GotoFrameCoords of current frame.
func (r *Rewind) GotoFrameCoords(scanline int, horizpos int) error {
	// frame to which to run the catch-up loop
	frame := r.vcs.TV.GetState(signal.ReqFramenum)

	// get nearest index of entry from which we can (re)generate the current frame
	idx, _, _ := r.findFrameIndex(frame)

	// if found index does not point to an immediately suitable state then try
	// the adhoc state if available
	if frame != r.entries[idx].TV.GetState(signal.ReqFramenum)+1 {
		if r.adhoc != nil && r.adhoc.TV.GetState(signal.ReqFramenum) == frame-1 {
			return r.plumbState(r.adhoc, frame, scanline, horizpos)
		}
	}

	// we've not used adhoc this time so nillify it
	r.adhoc = nil

	return r.plumb(idx, frame, scanline, horizpos)
}

// SetComparison points comparison to the most recent rewound entry.
func (r *Rewind) SetComparison() {
	r.comparison = r.entries[r.splice]
}

// GetComparison gets a reference to current comparison point.
func (r *Rewind) GetComparison() *State {
	return r.comparison
}

// NewFrame is in an implementation of television.FrameTrigger.
func (r *Rewind) NewFrame(_ bool) error {
	r.newFrame = true
	return nil
}

// Summary of the current state of the rewind system. The frame numbers for the
// snapshots at the start and end of the rewind history.
//
// Useful for GUIs for example, to present the range of frame numbers that are
// available in the rewind history.
//
// Note that there is no information about what type of snapshots the start and
// end frames are. This is intentional - I'm not sure that information would be
// useful.
type Summary struct {
	Start int
	End   int
}

func (r Rewind) GetSummary() Summary {
	e := r.end - 1
	if e < 0 {
		e += len(r.entries)
	}

	// because of how we generate visual state we cannot generate the image for
	// the first frame in the history unless the first entry represents a
	// machine reset
	//
	// this has a consequence when the first time the circular array wraps
	// around for the first time (the number of available entries drops by one)
	sf := r.entries[r.start].TV.GetState(signal.ReqFramenum)
	if r.entries[r.start].level != levelReset {
		sf++
	}

	return Summary{
		Start: sf,
		End:   r.entries[e].TV.GetState(signal.ReqFramenum),
	}
}
