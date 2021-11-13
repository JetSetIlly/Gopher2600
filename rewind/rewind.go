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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/logger"
)

// Runner provides the rewind package the opportunity to run the emulation.
type Runner interface {
	// CatchupLoop should loop until the frame/scanline/clock coordinates are
	// met. f should be called periodically, ideally every video cycle.
	//
	// When implementating the CatchupLoop(), care should be takan about what
	// to do for example, if the scaline/clock coordinates do no exist on the
	// specified frame. Either stop when the frame becomes too large or don't
	// request the rewind in the first place. Such details are outside the
	// scope of the rewind package however.
	CatchUpLoop(coords.TelevisionCoords) error
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

	// temporary entries should never appear in the history.
	levelTemporary
)

func (s State) String() string {
	switch s.level {
	case levelReset:
		return fmt.Sprintf("r(%d)", s.TV.GetCoords().Frame)
	case levelBoundary:
		return fmt.Sprintf("b(%d)", s.TV.GetCoords().Frame)
	case levelExecution:
		return fmt.Sprintf("e(%d)", s.TV.GetCoords().Frame)
	case levelTemporary:
		return fmt.Sprintf("t(%d)", s.TV.GetCoords().Frame)
	}
	return fmt.Sprintf("%d", s.TV.GetCoords().Frame)
}

// an overhead of two is required:
// (1) to accommodate the next index required for effective appending
// (2) we can't generate a screen for the first entry in the history, unless
// it's a reset entry, so we do not allow the rewind system to move to that
// frame.
const overhead = 2

// Rewind contains a history of machine states for the emulation.
type Rewind struct {
	emulation emulation.Emulation
	vcs       *hardware.VCS
	runner    Runner

	// optional timeline counter implementation
	ctr TimelineCounter

	// prefs for the rewind system
	Prefs *Preferences

	// timeline information. note that this is kept for convenience and sent as
	// a response to GetTimeline(). for internal package purposes the Start and
	// End fields are not useful and only updated when GetTimeline() is called.
	timeline Timeline

	// circular array of snapshotted entries. start and next indicate the
	// beginning and the end of the circular array. the next index can be
	// smaller than the start index
	entries []*State
	start   int
	next    int

	// the point at which new entries will be added
	splice int

	// pointer to the comparison point
	comparison *State

	// a new frame has been triggered. resolve as soon as possible.
	newFrame bool

	// the number frames since snapshot (not counting levelExecution
	// snapshots)
	framesSinceSnapshot int

	// a rewind boundary has been detected. call reset() on next frame.
	boundaryNextFrame bool
}

// NewRewind is the preferred method of initialisation for the Rewind type.
func NewRewind(emulation emulation.Emulation, runner Runner) (*Rewind, error) {
	r := &Rewind{
		emulation: emulation,
		vcs:       emulation.VCS().(*hardware.VCS),
		runner:    runner,
	}

	var err error

	r.Prefs, err = newPreferences(r)
	if err != nil {
		return nil, curated.Errorf("rewind: %v", err)
	}

	r.timeline = newTimeline()
	r.allocate()

	return r, nil
}

// AddTimelineCounter to the rewind system. Augments Timeline information that
// would otherwisde be awkward to gather.
//
// Only one timeline counter can be used at any one time (ie. subsequent calls
// to AddTimelineCounter() will override previous calls.)
func (r *Rewind) AddTimelineCounter(ctr TimelineCounter) {
	r.ctr = ctr
}

// initialise space for entries and reset rewind system.
func (r *Rewind) allocate() {
	r.entries = make([]*State, r.Prefs.MaxEntries.Get().(int)+overhead)
	r.reset(levelReset)
}

// Reset rewind system removes all entries and takes a snapshot of the
// execution state. This should be called whenever a new cartridge is attached
// to the emulation.
func (r *Rewind) Reset() {
	r.reset(levelReset)
}

// reset rewind system and use the specified snapshotLevel for the first entry.
// this will usually be levelReset but levelBoundary is also a sensible value.
//
// levelReset should really only be used when the vcs has actually been reset.
func (r *Rewind) reset(level snapshotLevel) {
	// nillify all entries
	for i := range r.entries {
		r.entries[i] = nil
	}

	r.comparison = nil

	r.newFrame = false
	r.framesSinceSnapshot = 0
	r.boundaryNextFrame = false

	// start and next equal to begin with. the first call to append() below
	// will add the new State at the current next index and then advance the
	// next index ready for the next append()
	r.start = 1
	r.next = 1

	// the splice point is checked to see if it is an execution entry and is
	// chopped off if it is. the insertion of a sparse boundary entry means we
	// don't have to check for nil
	//
	// the append function will move the splice index to start
	//
	// this arrangement of the stand, next and splice indexes means that there
	// are no special conditions in the append() function.
	r.splice = 0

	// the first entry is a boundary entry by definition
	r.entries[r.splice] = &State{level: levelBoundary}

	// add current state as first entry
	r.append(r.snapshot(level))

	// first comparison is to the snapshot of the reset machine
	r.comparison = r.entries[r.start]
}

func (r *Rewind) String() string {
	s := strings.Builder{}

	i := r.start
	for i < r.next && i < len(r.entries) {
		e := r.entries[i]
		if e != nil {
			s.WriteString(fmt.Sprintf("%s ", e.String()))
		}
		i++
	}

	if i != r.next {
		i = 0
		for i < r.next {
			e := r.entries[i]
			if e != nil {
				s.WriteString(fmt.Sprintf("%s ", e.String()))
			}
			i++
		}
	}

	return s.String()
}

// the index of the last entry in the circular rewind history to be written to.
// the end index points to the *next* entry to be written to.
func (r *Rewind) lastEntryIdx() int {
	e := r.next - 1
	if e < 0 {
		e += len(r.entries)
	}
	return e
}

// snapshot the supplied VCS instance.
func snapshot(vcs *hardware.VCS, level snapshotLevel) *State {
	return &State{
		level: level,
		CPU:   vcs.CPU.Snapshot(),
		Mem:   vcs.Mem.Snapshot(),
		RIOT:  vcs.RIOT.Snapshot(),
		TIA:   vcs.TIA.Snapshot(),
		TV:    vcs.TV.Snapshot(),
	}
}

// snapshot the 'current' VCS instance.
func (r *Rewind) snapshot(level snapshotLevel) *State {
	return snapshot(r.vcs, level)
}

// GetCurrentState returns a temporary snapshot of the current state.
//
// It is not added to the rewind history.
func (r *Rewind) GetCurrentState() *State {
	return r.snapshot(levelTemporary)
}

// RecordFrameState should be called after every CPU instruction to check
// whether a new frame has been triggered since the last call. Delaying a call
// to this function may result in sub-optimal results.
//
// Does nothing if called when the machine is mid CPU instruction.
func (r *Rewind) RecordFrameState() {
	if !r.vcs.CPU.LastResult.Final && !r.vcs.CPU.HasReset() {
		logger.Logf("rewind", "RecordFrameState() attempted mid CPU instruction")
		return
	}

	r.boundaryNextFrame = r.boundaryNextFrame || r.vcs.Mem.Cart.RewindBoundary()

	if !r.newFrame {
		return
	}
	r.newFrame = false

	if r.boundaryNextFrame {
		r.boundaryNextFrame = false
		r.reset(levelBoundary)
		logger.Logf("rewind", "boundary added at frame %d", r.vcs.TV.GetCoords().Frame)
		return
	}

	// add state only if frequency check passes
	r.framesSinceSnapshot++
	if r.framesSinceSnapshot%r.Prefs.Freq.Get().(int) != 0 {
		return
	}

	r.framesSinceSnapshot = 0

	r.append(r.snapshot(levelFrame))
}

// RecordExecutionState takes a snapshot of the emulation's ExecutionState
// state.
//
// Does nothing if called when the machine is mid CPU instruction.
func (r *Rewind) RecordExecutionState() {
	if !r.vcs.CPU.LastResult.Final && !r.vcs.CPU.HasReset() {
		logger.Logf("rewind", "RecordExecutionState() attempted mid CPU instruction")
		return
	}

	// no need to record the execution state if the current state is the same
	// as the most recent state recorded
	if !coords.Equal(r.vcs.TV.GetCoords(), r.entries[r.lastEntryIdx()].TV.GetCoords()) {
		r.append(r.snapshot(levelExecution))
	}
}

// append the state to the end of the list of entries. handles the splice
// point correctly and any forgetting of old states that have expired.
func (r *Rewind) append(s *State) {
	// chop off the end entry if it is in execution entry. we must do this
	// before any further appending. this is enough to ensure that there is
	// never more than one execution entry in the history.
	if r.entries[r.splice].level == levelExecution {
		r.next = r.splice
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
	r.next = r.splice + 1
	if r.next >= len(r.entries) {
		r.next = 0
	}

	// push start index along
	if r.next == r.start {
		r.start++
		if r.start >= len(r.entries) {
			r.start = 0
		}
	}
}

// plumb in state supplied as the argument
func plumb(vcs *hardware.VCS, state *State) {
	// tv plumbing works a bit different to other areas because we're only
	// recording the state of the TV not the entire TV itself.
	vcs.TV.PlumbState(vcs, state.TV.Snapshot())

	// take another snapshot of the state before plumbing. we don't want the
	// machine to change what we have stored in our state array (we learned
	// that lesson the hard way :-)
	vcs.CPU = state.CPU.Snapshot()
	vcs.Mem = state.Mem.Snapshot()
	vcs.RIOT = state.RIOT.Snapshot()
	vcs.TIA = state.TIA.Snapshot()

	vcs.CPU.Plumb(vcs.Mem)
	vcs.Mem.Plumb()
	vcs.RIOT.Plumb(vcs.Mem.RIOT, vcs.Mem.TIA)
	vcs.TIA.Plumb(vcs.TV, vcs.Mem.TIA, vcs.RIOT.Ports, vcs.CPU)

	// reset peripherals after new state has been plumbed. without this,
	// controllers can feel odd if the newly plumbed state has left RIOT memory
	// in a latched state
	vcs.RIOT.Ports.ResetPeripherals()
}

// run from the supplied state until the cooridinates are reached.
//
// note that this will not change the splice point or update the
// framesSinceSnapshot value. use setSplicePoint() for that
func (r *Rewind) runFromStateToCoords(fromState *State, toCoords coords.TelevisionCoords) error {
	plumb(r.vcs, fromState)

	// if this is a reset entry then TV must be reset
	if fromState.level == levelReset {
		err := r.vcs.TV.Reset(false)
		if err != nil {
			return curated.Errorf("rewind: %v", err)
		}
	}

	err := r.runner.CatchUpLoop(toCoords)
	if err != nil {
		return curated.Errorf("rewind: %v", err)
	}

	return nil
}

// setSplicePoint sets the splice point to the supplied index. the emulation
// will be run to the supplied frame, scanline, clock point.
func (r *Rewind) setSplicePoint(fromIdx int, toCoords coords.TelevisionCoords) error {
	r.splice = fromIdx
	fromState := r.entries[fromIdx]

	// plumb in selected entry
	err := r.runFromStateToCoords(fromState, toCoords)
	if err != nil {
		return err
	}

	return nil
}

// findFrameIndex returns a lot of information and so is wrapped in a
// findResults type
type findResults struct {
	fromIdx   int
	fromFrame int
	isFuture  bool
}

// find index nearest to the requested frame. returns the index and the frame
// number that is actually possible with the rewind system.
//
// the future value indicates that the requested frame is past the end of the
// history. if future is true then idx and frame will point to the most recent
// entry that we do have.
//
// note that findFrameIndex() searches for the frame that is two frames before
// the one that is requested.
func (r *Rewind) findFrameIndex(frame int) findResults {
	sf := frame - 1
	if r.emulation.Mode() == emulation.ModeDebugger {
		sf--
	}

	// initialise binary search
	s := r.start
	e := r.lastEntryIdx()

	// check whether request is out of bounds of the rewind history. if it is
	// then plumb in the nearest entry

	// is requested frame too old (ie. before the start of the array)
	fn := r.entries[s].TV.GetCoords().Frame
	if sf < fn {
		return findResults{fromIdx: s, fromFrame: fn}
	}

	// is requested frame too new (ie. past the end of the array)
	fn = r.entries[e].TV.GetCoords().Frame
	if frame > fn {
		return findResults{fromIdx: e, fromFrame: fn, isFuture: true}
	}

	// because r.entries is a cirular array, there's an additional step to the
	// binary search. if start (lower) is greater then end (upper) then check
	// which half of the circular array to concentrate on.
	if r.start > e {
		fn := r.entries[len(r.entries)-1].TV.GetCoords().Frame
		if sf <= fn {
			e = len(r.entries) - 1
		} else {
			e = r.start - 1
			s = 0
		}
	}

	// the range which we must consider to be a match
	freqAdj := r.Prefs.Freq.Get().(int) - 1

	// normal binary search
	for s <= e {
		idx := (s + e) / 2

		fn := r.entries[idx].TV.GetCoords().Frame

		// check for match, taking into consideration the gaps introduced by
		// the frequency value
		if sf >= fn && sf <= fn+freqAdj {
			return findResults{fromIdx: idx, fromFrame: fn}
		}

		if sf < fn {
			e = idx - 1
		}
		if sf > fn {
			s = idx + 1
		}
	}

	logger.Logf("rewind", "cannot find frame %d in the rewind history", frame)
	return findResults{fromIdx: e, fromFrame: r.entries[e].TV.GetCoords().Frame}
}

// RerunLastNFrames runs the emulation from the a point N frames in the past to
// the current state.
func (r *Rewind) RerunLastNFrames(frames int) error {
	to := r.GetCurrentState()
	ff := to.TV.GetCoords().Frame
	if ff < 0 {
		ff = 0
	}
	idx := r.findFrameIndex(ff).fromIdx

	err := r.setSplicePoint(idx, to.TV.GetCoords())
	if err != nil {
		return curated.Errorf("rewind: %v", err)
	}

	return nil
}

// GotoCoords moves emulation to specified frame/scanline/clock "coordinates".
func (r *Rewind) GotoCoords(toCoords coords.TelevisionCoords) error {
	// get nearest index of entry from which we can being to (re)generate the
	// current frame
	res := r.findFrameIndex(toCoords.Frame)

	if res.isFuture {
		toCoords.Frame = res.fromFrame
	}

	err := r.setSplicePoint(res.fromIdx, toCoords)
	if err != nil {
		return err
	}

	return nil

}

// GotoLast goes to the last entry in the rewind history. It handles situations
// when the last entry is an execution state.
func (r *Rewind) GotoLast() error {
	e := r.lastEntryIdx()

	toCoords := r.entries[e].TV.GetCoords()

	// goto the beginning of the frame if last entry is not an execution frame
	if r.entries[e].level != levelExecution {
		toCoords.Scanline = 0
		toCoords.Clock = -specification.ClksHBlank
	}

	// make adjustments to the index so we call setSplicePoint from a suitable place.
	e -= 2
	if e < 0 {
		e += len(r.entries)
	}

	// boundary checks to make sure we haven't gone back past the beginning of
	// the circular array. this can happen if a REWIND LAST command, for
	// example, is issued before any history has been recorded
	if r.entries[e] == nil {
		e = r.start
	}

	return r.setSplicePoint(e, toCoords)
}

// GotoFrame is a special case of GotoCoords that requires the frame number only.
func (r *Rewind) GotoFrame(frame int) error {
	return r.GotoCoords(coords.TelevisionCoords{Frame: frame, Clock: -specification.ClksHBlank})
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
func (r *Rewind) NewFrame(frameInfo television.FrameInfo) error {
	r.addTimelineEntry(frameInfo)
	r.newFrame = true
	return nil
}
