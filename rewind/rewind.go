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

// CatchUpLoopCallback is called by Runner.CatchUpLoop() implementations. The
// rewind package will use this to keep the rewind state crisp.
type CatchUpLoopCallback func(fr int)

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
	CatchUpLoop(coords.TelevisionCoords, CatchUpLoopCallback) error
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
	return fmt.Sprintf("%d", s.TV.GetCoords().Frame)
}

// an overhead of two is required. (1) to accommodate the end index required for
// effective appending; (2) we can't generate a screen for the first entry in
// the history, unless it's a reset entry, so we do not allow the rewind system
// to move to that frame.
const overhead = 2

// Rewind contains a history of machine states for the emulation.
type Rewind struct {
	vcs    *hardware.VCS
	ctr    TimelineCounter
	runner Runner

	// state of emulation
	emulationState emulation.State

	// prefs for the rewind system
	Prefs *Preferences

	// timeline information. note that this is kept for convenience and sent as
	// a response to GetTimeline(). for internal package purposes the Start and
	// End fields are not useful and only updated when GetTimeline() is called.
	timeline Timeline

	// circular arry of snapshotted entries. start and end indicate the first
	// and last index. the last index can be smaller the start index
	entries []*State
	start   int
	end     int

	// the point at which new entries will be added
	splice int

	// pointer to the comparison point
	comparison *State

	// adhocFrame is a special snapshot of a state that cannot be found in the
	// entries array. it is used to speed up consecutive calls to GotoCoords()
	//
	// only comes into play if snapshot frequency is larger than 1
	adhocFrame *State

	// a new frame has been triggered. resolve as soon as possible.
	newFrame bool

	// a snapshot has just been added by the Check() function. we use this to
	// prevent another snapshot being taken by ExecutionState(). rarely comes
	// into play but it prevents what would essentially be a duplicate entry
	// being added.
	justAddedLevelFrame bool

	// the number frames since snapshot (not counting levelExecution
	// snapshots)
	framesSinceSnapshot int

	// a rewind boundary has been detected. call reset() on next frame.
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
		return nil, curated.Errorf("rewind: %v", err)
	}

	r.timeline = newTimeline()
	r.allocate()

	return r, nil
}

// SetEmulationState is called by emulation whenever state changes. How we
// handle the rewind depends on the current state.
func (r *Rewind) SetEmulationState(state emulation.State) {
	r.emulationState = state
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

	r.adhocFrame = nil
	r.comparison = nil

	r.newFrame = false
	r.justAddedLevelFrame = true
	r.framesSinceSnapshot = 0
	r.boundaryNextFrame = false

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

// GetCurrentState creates a returns an adhoc snapshot of the current state. It does
// not add the state to the rewind history.
func (r *Rewind) GetCurrentState() *State {
	return r.snapshot(levelAdhoc)
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
		r.justAddedLevelFrame = false
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

	r.justAddedLevelFrame = true
	r.framesSinceSnapshot = 0

	r.append(r.snapshot(levelFrame))
}

// RecordExecutionState takes a snapshot of the emulation's ExecutionState state. It
// will do nothing if the last call to ResolveNewFrame() resulted in a snapshot
// being taken.
//
// Does nothing if called when the machine is mid CPU instruction.
func (r *Rewind) RecordExecutionState() {
	if !r.vcs.CPU.LastResult.Final && !r.vcs.CPU.HasReset() {
		logger.Logf("rewind", "RecordExecutionState() attempted mid CPU instruction")
		return
	}

	if !r.justAddedLevelFrame {
		r.append(r.snapshot(levelExecution))
	}
}

// append the state to the end of the list of entries. handles  the splice
// point correctly and any forgetting of old states that have expired.
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

	// splice timeline at current frame number
	r.timeline.splice(r.vcs.TV.GetCoords().Frame)
}

// setContinuePoint sets the splice point to the supplied index. the emulation
// will be run to the supplied frame, scanline, clock point.
func (r *Rewind) setContinuePoint(idx int, coords coords.TelevisionCoords) error {
	// current index is the index we're plumbing in. this has nothing to do
	// with the frame number (especially important to remember if frequency is
	// greater than 1)
	r.splice = idx

	s := r.entries[idx]

	// plumb in selected entry
	err := r.plumbState(s, coords)
	if err != nil {
		return err
	}

	return nil
}

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
}

// plumb in state supplied as the argument. catch-up loop will halt as soon as
// possible after frame/scanline/clock is reached or surpassed
//
// note that this will not update the splice point up update the framesSinceSnapshot
// value. use plumb() with an index into the history for that.
func (r *Rewind) plumbState(s *State, coords coords.TelevisionCoords) error {
	plumb(r.vcs, s)

	// if this is a reset entry then TV must be reset
	if s.level == levelReset {
		err := r.vcs.TV.Reset(false)
		if err != nil {
			return curated.Errorf("rewind: %v", err)
		}
	}

	// snapshot adhoc frame as soon as convenient. not required when snapshot
	// frequency is one
	adhoc := r.Prefs.Freq.Get().(int) == 1

	callback := func(fr int) {
		if !adhoc && fr == coords.Frame-1 {
			// only make an adhoc snapshot on an instruction boundary. if we
			// don't check for this then we risk saving an adhoc state that
			// will immediately crash when it's plumbed in
			if r.vcs.CPU.LastResult.Final {
				r.adhocFrame = r.snapshot(levelAdhoc)
				adhoc = true
			}
		}
	}

	err := r.runner.CatchUpLoop(coords, callback)
	if err != nil {
		return curated.Errorf("rewind: %v", err)
	}

	return nil
}

// GotoLast sets the emulation state to the most recent entry.
func (r *Rewind) GotoLast() error {
	idx := r.end - 1
	if idx < 0 {
		idx += len(r.entries)
	}

	coords := r.entries[idx].TV.GetCoords()

	// got ot the beginning of the frame if last entry is not an execution frame
	if r.entries[idx].level != levelExecution {
		coords.Scanline = 0
		coords.Clock = -specification.ClksHBlank
	}

	// make adjustments to the index so we plumbing from a suitable place
	idx -= 2
	if idx < 0 {
		idx += len(r.entries)
	}

	// boundary checks to make sure we haven't gone back past the beginning of
	// the circular array
	if r.entries[idx] == nil {
		idx = r.start
	}

	return r.setContinuePoint(idx, coords)
}

// GotoFrame searches the rewind history for the frame number. If the precise
// frame number can not be found the nearest frame will be plumbed in and the
// emulation run to match the requested frame.
func (r *Rewind) GotoFrame(frame int) error {
	idx, frame, last := r.findFrameIndex(frame)

	coords := coords.TelevisionCoords{
		Frame:    frame,
		Scanline: 0,
		Clock:    -specification.ClksHBlank,
	}

	// it is more appropriate to plumb with GotoLast() if last is true
	if last {
		return r.GotoLast()
	}

	return r.setContinuePoint(idx, coords)
}

// find index nearest to the requested frame. returns the index and the frame
// number that is actually possible with the rewind system.
//
// the last value indicates that the requested frame is past the end of the
// history. in those instances, the returned frame number can be used for the
// plumbing operation or because last==true the GotoLast() can be used for a
// more natural feeling result.
//
// note that findFrameIndex() searches for the frame that is two frames before
// the one that is requested.
func (r *Rewind) findFrameIndex(frame int) (idx int, fr int, last bool) {
	sf := frame
	switch sf {
	case 0:
		sf = 0
	case 1:
		sf = 0
	default:
		sf -= 2
	}

	// initialise binary search
	s := r.start
	e := r.end - 1
	if e < 0 {
		e += len(r.entries)
	}

	// check whether request is out of bounds of the rewind history. if it is
	// then plumb in the nearest entry

	// is requested frame too old (ie. before the start of the array)
	fn := r.entries[s].TV.GetCoords().Frame
	if sf < fn {
		return s, fn + 1, false
	}

	// is requested frame too new (ie. past the end of the array)
	fn = r.entries[e].TV.GetCoords().Frame
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
			return idx, frame, false
		}

		if sf < fn {
			e = idx - 1
		}
		if sf > fn {
			s = idx + 1
		}
	}

	logger.Logf("rewind", "cannot find frame %d in the rewind history", frame)
	return e, frame, false
}

// GotoState will the run the VCS to the quoted state.
func (r *Rewind) GotoState(state *State) error {
	return r.GotoCoords(state.TV.GetCoords())
}

type PokeHook func(res *State) error

// RunFromState will the run the VCS from one state to another state.
func (r *Rewind) RunFromState(from *State, to *State, poke PokeHook) error {
	ff := from.TV.GetCoords().Frame
	idx, _, _ := r.findFrameIndex(ff)

	if poke != nil {
		err := poke(r.entries[idx])
		if err != nil {
			return err
		}
	}

	err := r.setContinuePoint(idx, to.TV.GetCoords())
	if err != nil {
		return curated.Errorf("rewind: %v", err)
	}

	return nil
}

// RerunLastNFrames runs the emulation from the a point N frames in the past to
// the current state.
func (r *Rewind) RerunLastNFrames(frames int) error {
	to := r.GetCurrentState()
	ff := to.TV.GetCoords().Frame
	if ff < 0 {
		ff = 0
	}
	idx, _, _ := r.findFrameIndex(ff)

	err := r.setContinuePoint(idx, to.TV.GetCoords())
	if err != nil {
		return curated.Errorf("rewind: %v", err)
	}

	return nil
}

// GotoCoords moves emulation to specified frame/scanline/clock "coordinates".
func (r *Rewind) GotoCoords(coords coords.TelevisionCoords) error {
	// get nearest index of entry from which we can being to (re)generate the
	// current frame
	idx, _, _ := r.findFrameIndex(coords.Frame)

	// if found index does not point to an immediately suitable state then try
	// the adhocFrame state if available
	if coords.Frame != r.entries[idx].TV.GetCoords().Frame+1 {
		if r.adhocFrame != nil && r.adhocFrame.TV.GetCoords().Frame == coords.Frame-1 {
			return r.plumbState(r.adhocFrame, coords)
		}
	}

	// we've not used adhoc this time so nillify it
	r.adhocFrame = nil

	err := r.setContinuePoint(idx, coords)
	if err != nil {
		return err
	}

	return nil
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
