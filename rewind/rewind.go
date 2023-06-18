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

	"github.com/jetsetilly/gopher2600/debugger/govern"
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

// Emulation defines as much of the emulation we require access to.
type Emulation interface {
	Mode() govern.Mode
	State() govern.State
	VCS() *hardware.VCS
}

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

func (s *State) snapshot() *State {
	return &State{
		level: s.level,
		CPU:   s.CPU.Snapshot(),
		Mem:   s.Mem.Snapshot(),
		RIOT:  s.RIOT.Snapshot(),
		TIA:   s.TIA.Snapshot(),
		TV:    s.TV.Snapshot(),
	}
}

// snapshotLevel indicates the level of snapshot.
type snapshotLevel int

// List of valid SnapshotLevel values.
const (
	// reset and boundary entries should only even appear once at the start of the history
	levelReset snapshotLevel = iota
	levelBoundary

	// a frame entry is a recording on a frame boundary (as soon as possible
	// after a new frame). when they are made is based on the current snapshot
	// frequency.
	//
	// there can be many frame entries the rewind history.
	levelFrame

	// execution entries only ever appear once at the end of the history.
	// moreover they only ever appear when the snapshot frequency is greater
	// than one.
	levelExecution

	// temporary entries should never appear in the history.
	levelTemporary
)

func (s *State) String() string {
	if s == nil {
		return "----"
	}
	switch s.level {
	case levelReset:
		return fmt.Sprintf("r%03d", s.TV.GetCoords().Frame)
	case levelBoundary:
		return fmt.Sprintf("b%03d", s.TV.GetCoords().Frame)
	case levelExecution:
		return fmt.Sprintf("e%03d", s.TV.GetCoords().Frame)
	case levelTemporary:
		return fmt.Sprintf("t%03d", s.TV.GetCoords().Frame)
	}
	return fmt.Sprintf("f%03d", s.TV.GetCoords().Frame)
}

// an overhead of two is required:
// (1) to accommodate the next index required for effective appending
// (2) we can't generate a screen for the first entry in the history, unless
// it's a reset entry, so we do not allow the rewind system to move to that
// frame.
const overhead = 2

// Rewind contains a history of machine states for the emulation.
type Rewind struct {
	emulation Emulation
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
	comparison       *State
	comparisonLocked bool

	// coordinates of execution state
	executionCoords coords.TelevisionCoords

	// a new frame has been triggered. resolve as soon as possible.
	newFrame bool

	// a rewind boundary has been detected. call reset() on next frame.
	boundaryNextFrame bool
}

// NewRewind is the preferred method of initialisation for the Rewind type.
func NewRewind(emulation Emulation, runner Runner) (*Rewind, error) {
	r := &Rewind{
		emulation: emulation,
		vcs:       emulation.VCS(),
		runner:    runner,
	}

	var err error

	r.Prefs, err = newPreferences(r)
	if err != nil {
		return nil, fmt.Errorf("rewind: %w", err)
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
// execution state. Resets timeline too.
//
// This should be called whenever a new cartridge is attached to the emulation.
func (r *Rewind) Reset() {
	r.reset(levelReset)
	r.timeline.reset()
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
	r.boundaryNextFrame = false

	// reset start and next
	r.start = 0
	r.next = 1

	// the splice point is checked to see if it is an execution entry and is
	// chopped off if it is. the insertion of a sparse boundary entry means we
	// don't have to check for nil
	//
	// the append function will move the splice index to start
	//
	// this arrangement of the stand, next and splice indexes means that there
	// are no special conditions in the append() function.
	r.splice = len(r.entries) - 1

	// add current state as first entry
	r.append(r.snapshot(level))

	// and as the second entry
	r.append(r.snapshot(levelFrame))

	// first comparison is to the snapshot of the reset machine
	r.comparison = r.entries[r.start]

}

// String outputs the entry information for the entire rewind history. The
// Peephole() funcion is probably a better option.
func (r *Rewind) String() string {
	s := strings.Builder{}

	if r.start < r.next {
		for i := r.start; i < r.next; i++ {
			e := r.entries[i]
			if e != nil {
				s.WriteString(fmt.Sprintf("%s ", e.String()))
			}
		}
	} else {
		for i := r.start; i < len(r.entries); i++ {
			e := r.entries[i]
			if e != nil {
				s.WriteString(fmt.Sprintf("%s ", e.String()))
			}
		}
		for i := 0; i < r.next; i++ {
			e := r.entries[i]
			if e != nil {
				s.WriteString(fmt.Sprintf("%s ", e.String()))
			}
		}
	}

	return s.String()
}

// Peephole outputs a short summary of the state of the rewind system centered
// on the current splice value
func (r *Rewind) Peephole() string {
	const peephole = 5

	var split bool
	peepi := r.splice - peephole
	if peepi < 0 {
		peepi += len(r.entries)
		split = true
	}
	peepj := r.splice + peephole
	if peepj >= len(r.entries) {
		peepj -= len(r.entries)
		if split {
			panic("length of entries in rewind is too short")
		}
		split = true
	}

	// build output string
	b := strings.Builder{}

	f := func(i, j int) {
		for k, e := range r.entries[i:j] {
			if k+i == r.splice {
				b.WriteString(fmt.Sprintf("(%s) ", e))
			} else {
				b.WriteString(fmt.Sprintf("%s ", e))
			}
		}
	}

	b.WriteString(fmt.Sprintf("[%03d] ", peepi))
	if split {
		f(peepi, len(r.entries))
		b.WriteString(fmt.Sprintf("| "))
		f(0, peepj)
	} else {
		b.WriteString("  ")
		f(peepi, peepj)
	}
	b.WriteString(fmt.Sprintf("[%03d]\n", peepj))

	return b.String()
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

// RecordState should be called after every CPU instruction. A new state will
// be recorded if the current rewind policy agrees.
func (r *Rewind) RecordState() {
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

	fn := r.vcs.TV.GetCoords().Frame
	if fn%r.Prefs.Freq.Get().(int) != 0 {
		r.append(r.snapshot(levelExecution))
		return
	}

	r.append(r.snapshot(levelFrame))
}

// RecordExecutionCoords records the coordinates of the current execution state.
func (r *Rewind) RecordExecutionCoords() {
	r.executionCoords = r.vcs.TV.GetCoords()
}

// append the state to the end of the list of entries. handles the splice
// point correctly and any forgetting of old states that have expired.
func (r *Rewind) append(s *State) {
	// chop off the end entry if it is in execution entry. we must do this
	// before any further appending. this is enough to ensure that there is
	// never more than one execution entry in the history.
	if r.entries[r.splice] != nil && r.entries[r.splice].level == levelExecution {
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

// Plumb state into VCS. The fromDifferentEmulation indicates that the State
// has been created by a different VCS instance than the one being plumbed into.
func Plumb(vcs *hardware.VCS, state *State, fromDifferentEmulation bool) {
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

	// finish off plumbing process
	vcs.Plumb(fromDifferentEmulation)
}

// run from the supplied state until the cooridinates are reached.
//
// note that this will not change the splice point. use setSplicePoint() for that
func (r *Rewind) runFromStateToCoords(fromState *State, toCoords coords.TelevisionCoords) error {
	Plumb(r.vcs, fromState, false)

	// if this is a reset entry then TV must be reset
	if fromState.level == levelReset {
		err := r.vcs.TV.Reset(false)
		if err != nil {
			return fmt.Errorf("rewind: %w", err)
		}
	}

	err := r.runner.CatchUpLoop(toCoords)
	if err != nil {
		return fmt.Errorf("rewind: %w", err)
	}

	return nil
}

// setSplicePoint sets the splice point to the supplied index. the emulation
// will be run to the supplied frame, scanline, clock point.
func (r *Rewind) setSplicePoint(fromIdx int, toCoords coords.TelevisionCoords) error {
	// set new splice point
	r.splice = fromIdx + 1
	if r.splice >= len(r.entries) {
		r.splice -= len(r.entries)
	}

	// plumb in selected entry
	fromState := r.entries[fromIdx]
	err := r.runFromStateToCoords(fromState, toCoords)
	if err != nil {
		return err
	}

	return nil
}

// findFrameIndex returns a lot of information and so is wrapped in a
// findResults type
type findResults struct {
	nearestIdx   int
	nearestFrame int
	future       bool
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
	frame--
	if r.emulation.Mode() == govern.ModeDebugger {
		frame--
	}
	return r.findFrameIndexExact(frame)
}

// find index of requested frame
func (r *Rewind) findFrameIndexExact(frame int) findResults {
	// initialise binary search
	s := r.start
	e := r.lastEntryIdx()

	// check whether request is out of bounds of the rewind history. if it is
	// then plumb in the nearest entry

	// is requested frame too old (ie. before the start of the array)
	fn := r.entries[s].TV.GetCoords().Frame
	if frame < fn {
		return findResults{nearestIdx: s, nearestFrame: fn}
	}

	// is requested frame too new (ie. past the end of the array)
	fn = r.entries[e].TV.GetCoords().Frame
	if frame > fn {
		return findResults{nearestIdx: e, nearestFrame: fn, future: true}
	}
	// the range which we must consider to be a match
	freqAdj := r.Prefs.Freq.Get().(int) - 1

	// because r.entries is a cirular array, there's an additional step to the
	// binary search. if start (lower) is greater then end (upper) then check
	// which half of the circular array to concentrate on.
	if s > e {
		fn := r.entries[len(r.entries)-1].TV.GetCoords().Frame
		if frame <= fn+freqAdj {
			e = len(r.entries) - 1
		} else {
			e = r.start - 1
			s = 0
		}
	}

	// normal binary search
	for s <= e {
		idx := (s + e) / 2

		fn := r.entries[idx].TV.GetCoords().Frame

		// check for match, taking into consideration the gaps introduced by
		// the frequency value
		if frame >= fn && frame <= fn+freqAdj {
			return findResults{nearestIdx: idx, nearestFrame: fn}
		}

		if frame < fn {
			e = idx - 1
		}
		if frame > fn {
			s = idx + 1
		}
	}

	panic(fmt.Sprintf("rewind: cannot find frame %d in the rewind history", frame))
}

// RerunLastNFrames runs the emulation from the a point N frames in the past to
// the current state.
func (r *Rewind) RerunLastNFrames(frames int) error {
	to := r.GetCurrentState()
	ff := to.TV.GetCoords().Frame
	if ff < 0 {
		ff = 0
	}

	idx := r.findFrameIndex(ff).nearestIdx
	err := r.setSplicePoint(idx, to.TV.GetCoords())
	if err != nil {
		return fmt.Errorf("rewind: %w", err)
	}

	return nil
}

// GotoCoords moves emulation to specified frame/scanline/clock "coordinates".
func (r *Rewind) GotoCoords(toCoords coords.TelevisionCoords) error {
	// get nearest index of entry from which we can being to (re)generate the
	// current frame
	res := r.findFrameIndex(toCoords.Frame)

	if res.future {
		toCoords = r.entries[res.nearestIdx].TV.GetCoords()
	}

	idx := res.nearestIdx
	err := r.setSplicePoint(idx, toCoords)
	if err != nil {
		return err
	}

	return nil

}

// GotoLast goes to the last entry in the rewind history. It handles situations
// when the last entry is an execution state.
func (r *Rewind) GotoLast() error {
	return r.GotoCoords(r.executionCoords)
}

// GotoFrame is a special case of GotoCoords that requires the frame number only.
func (r *Rewind) GotoFrame(frame int) error {
	return r.GotoCoords(coords.TelevisionCoords{Frame: frame, Clock: -specification.ClksHBlank})
}

// UpdateComparison points comparison to the current state
func (r *Rewind) UpdateComparison() {
	if r.comparisonLocked {
		return
	}
	r.comparison = r.GetCurrentState()
}

// SetComparison points comparison to the supplied state
func (r *Rewind) SetComparison(frame int) {
	res := r.findFrameIndexExact(frame)
	s := r.entries[res.nearestIdx]
	if s != nil {
		r.comparison = s.snapshot()
	}
}

// LockComparison stops the comparison point from being updated
func (r *Rewind) LockComparison(locked bool) {
	r.comparisonLocked = locked
}

// NewFrame is in an implementation of television.FrameTrigger.
func (r *Rewind) NewFrame(frameInfo television.FrameInfo) error {
	r.addTimelineEntry(frameInfo)
	r.newFrame = true
	return nil
}

// GetState returns a copy for the nearest state for the indicated frame.
func (r *Rewind) GetState(frame int) *State {
	// get nearest index of entry from which we can being to (re)generate the
	// current frame
	res := r.findFrameIndex(frame)
	s := r.entries[res.nearestIdx]

	// return copy of state
	return s.snapshot()
}

// GetCurrentState returns a temporary snapshot of the current state.
func (r *Rewind) GetCurrentState() *State {
	return r.snapshot(levelTemporary)
}

// ComparisonState is returned by GetComparisonState()
type ComparisonState struct {
	State  *State
	Locked bool
}

// GetComparisonState gets a reference to current comparison point. This is not
// a copy of the state but the actual state. Also returns the
func (r *Rewind) GetComparisonState() ComparisonState {
	return ComparisonState{
		State:  r.comparison.snapshot(),
		Locked: r.comparisonLocked,
	}
}
