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

package television

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// the number of additional lines over the NTSC spec that is allowed before the
// TV flips to the PAL specification.
const excessScanlinesNTSC = 40

// the number of synced frames where we can expect things to be in flux.
const leadingFrames = 5

// the number of synced frames required before the tv frame is considered to "stable".
const stabilityThreshold = 20

// State encapsulates the television values that can change from moment to
// moment. Used by the rewind system when recording the current television
// state.
type State struct {
	// television specification (NTSC or PAL)
	spec specification.Spec

	// auto flag indicates that the tv type/specification should switch if it
	// appears to be outside of the current spec.
	//
	// in practice this means that if auto is true then we start with the NTSC
	// spec and move to PAL if the number of scanlines exceeds the NTSC maximum
	auto bool

	// state of the television
	//	- the current color clock. the horizontal position where the next pixel
	//	will be drawn. also used to check we're receiving the correct signals
	//	at the correct time.
	clock int
	//	- the current frame
	frameNum int
	//	- the current scanline number
	scanline int

	// is current frame as a result of a VSYNC flyback or not (a "natural"
	// flyback). we use this in the context of newFrame() so we should probably
	// think of this as the previous frame.
	syncedFrame bool

	// the number of frames that have been formed in sequence because of a
	// "synced frame". we use this to decide:
	//
	//   * whether the image is "stable"
	//   * whether specification changes should still occur
	//
	// once stableFrames has reached the stabilityThreshold then it is never
	// reset to zero (except through an explicit call to Reset()).
	stableFrames int

	// record of signal attributes from the last call to Signal()
	lastSignal signal.SignalAttributes

	// vsyncCount records the number of consecutive clocks the vsync signal
	// has been sustained. we use this to help correctly implement vsync.
	vsyncCount int

	// top and bottom of screen as detected by vblank/color signal
	top    int
	bottom int

	// frame resizer
	resizer resizer

	// the frame/scanline/clock of the last CPU instruction
	boundaryClock    int
	boundaryFrameNum int
	boundaryScanline int
}

func (s *State) String() string {
	// I would like to include the lastSignal string in this too but I'm
	// leaving it out for now because of existing video regression entries with
	// TV state will fail with it added.
	//
	// !!TODO: consider adding lastSignal information to TV state string.
	return fmt.Sprintf("FR=%04d SL=%03d CL=%03d", s.frameNum, s.scanline, s.clock-specification.ClksHBlank)
}

// Snapshot makes a copy of the television state.
func (s *State) Snapshot() *State {
	n := *s
	return &n
}

// Returns state information. Not that ReqClock counts from
// "-specifcation.ClksHblank" and not zero as you might expect.
func (s *State) GetState(request signal.StateReq) int {
	switch request {
	case signal.ReqFramenum:
		return s.frameNum
	case signal.ReqScanline:
		return s.scanline
	case signal.ReqClock:
		return s.clock - specification.ClksHBlank
	}
	panic(fmt.Sprintf("television: unhandled tv state request (%v)", request))
}

// Television is a Television implementation of the Television interface. In all
// honesty, it's most likely the only implementation required.
type Television struct {
	// vcs will be nil unless AttachVCS() has been called
	vcs VCSReturnChannel

	// spec on creation ID is the string that was to ID the television
	// type/spec on creation. because the actual spec can change, the ID field
	// of the Spec type can not be used for things like regression
	// test recreation etc.
	reqSpecID string

	// framerate limiter
	lmtr limiter

	// list of PixelRenderer implementations to consult
	renderers []PixelRenderer

	// list of FrameTrigger implementations to consult
	frameTriggers []FrameTrigger

	// list of PauseTrigger implementations to consult
	pauseTriggers []PauseTrigger

	// list of audio mixers to consult
	mixers []AudioMixer

	// instance of current state (as supported by the rewind system)
	state *State

	// list of signals sent to pixel renderers since the beginning of the
	// current frame
	signals []signal.SignalAttributes

	// the index to write the next signal
	currentIdx int

	// the max index from the last frame
	lastMaxIdx int

	// pause forwarding of signals to pixel renderers
	pauseRendering bool
}

// NewReference creates a new instance of the reference television type,
// satisfying the Television interface.
func NewTelevision(spec string) (*Television, error) {
	tv := &Television{
		reqSpecID: strings.ToUpper(spec),
		state:     &State{},
		signals:   make([]signal.SignalAttributes, MaxSignalHistory),
	}

	// initialise frame rate limiter
	tv.lmtr.init(tv)
	tv.SetFPS(-1)

	// set specification
	err := tv.SetSpec(spec)
	if err != nil {
		return nil, err
	}

	// empty list of renderers
	tv.renderers = make([]PixelRenderer, 0)

	return tv, nil
}

func (tv *Television) String() string {
	return tv.state.String()
}

// Reset the television to an initial state.
func (tv *Television) Reset(keepFrameNum bool) error {
	// we definitely do not call this on television initialisation because the
	// rest of the system may not be yet be in a suitable state

	// we're no longer resetting the TV spec on Reset(). doing so interferes
	// with the flexibility required to set the spec based on filename settings
	// etc.

	if !keepFrameNum {
		tv.state.frameNum = 0
	}

	tv.state.clock = 0
	tv.state.scanline = 0
	tv.state.stableFrames = 0
	tv.state.syncedFrame = false
	tv.state.vsyncCount = 0
	tv.state.lastSignal = signal.SignalAttributes{}

	for i := range tv.signals {
		tv.signals[i] = signal.SignalAttributes{}
	}
	tv.currentIdx = 0
	tv.lastMaxIdx = len(tv.signals) - 1

	for _, r := range tv.renderers {
		r.Resize(tv.state.spec, tv.state.spec.AtariSafeTop, tv.state.spec.AtariSafeBottom)
		r.Reset()
	}

	for _, m := range tv.mixers {
		m.Reset()
	}

	return nil
}

// Snapshot makes a copy of the television state.
func (tv *Television) Snapshot() *State {
	return tv.state.Snapshot()
}

// PlumbState attaches an existing television state.
func (tv *Television) PlumbState(vcs VCSReturnChannel, s *State) {
	if s == nil {
		panic("television: cannot plumb in a nil state")
	}

	tv.state = s

	// make sure vcs knows about current spec
	tv.vcs = vcs
	if tv.vcs != nil {
		_ = tv.vcs.SetClockSpeed(tv.state.spec.ID)
	}

	// reset signal history
	tv.currentIdx = 0
	tv.lastMaxIdx = 0

	// resize renderers to match current state
	for _, r := range tv.renderers {
		_ = r.Resize(tv.state.spec, tv.state.top, tv.state.bottom)
	}
}

// AttachVCS attaches an implementation of the VCSReturnChannel.
func (tv *Television) AttachVCS(vcs VCSReturnChannel) {
	tv.vcs = vcs

	// notify the newly attached console of the current TV spec
	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(tv.state.spec.ID)
	}
}

// AddPixelRenderer registers an implementation of PixelRenderer. Multiple
// implemntations can be added.
func (tv *Television) AddPixelRenderer(r PixelRenderer) {
	tv.renderers = append(tv.renderers, r)
	tv.frameTriggers = append(tv.frameTriggers, r)
}

// AddFrameTrigger registers an implementation of FrameTrigger. Multiple
// implemntations can be added.
func (tv *Television) AddFrameTrigger(f FrameTrigger) {
	tv.frameTriggers = append(tv.frameTriggers, f)
}

// AddPauseTrigger registers an implementation of PauseTrigger.
func (tv *Television) AddPauseTrigger(p PauseTrigger) {
	tv.pauseTriggers = append(tv.pauseTriggers, p)
}

// AddAudioMixer registers an implementation of AudioMixer. Multiple
// implemntations can be added.
func (tv *Television) AddAudioMixer(m AudioMixer) {
	tv.mixers = append(tv.mixers, m)
}

// some televisions may need to conclude and/or dispose of resources
// gently. implementations of End() should call EndRendering() and
// EndMixing() on each PixelRenderer and AudioMixer that has been added.
//
// for simplicity, the Television should be considered unusable
// after EndRendering() has been called.
func (tv *Television) End() error {
	var err error

	// call new frame for all renderers
	for _, r := range tv.renderers {
		err = r.EndRendering()
	}

	// flush audio for all mixers
	for _, m := range tv.mixers {
		err = m.EndMixing()
	}

	return err
}

// Signal updates the current state of the television.
func (tv *Television) Signal(sig signal.SignalAttributes) error {
	// examine signal for resizing possibility
	tv.state.resizer.examine(tv, sig)

	// a Signal() is by definition a new color clock. increase the horizontal count
	tv.state.clock++

	// once we reach the scanline's back-porch we'll reset the clock counter
	// and wait for the HSYNC signal. we do this so that the front-porch and
	// back-porch are 'together' at the beginning of the scanline. this isn't
	// strictly technically correct but it's convenient to think about
	// scanlines in this way (rather than having a split front and back porch)
	if tv.state.clock >= specification.ClksScanline {
		tv.state.clock = 0

		// bump scanline counter
		tv.state.scanline++

		// reached end of screen without VSYNC sequence
		if tv.state.scanline > tv.state.spec.ScanlinesTotal {
			// fly-back naturally if VBlank is off. a good example of a ROM
			// that requires correct handling of this is Andrew Davies' Chess
			// (3e+ test rom)
			//
			// (06/01/21) another example is the Artkaris NTSC version of Lili
			if tv.state.scanline > specification.AbsoluteMaxScanlines {
				err := tv.newFrame(false)
				if err != nil {
					return err
				}
			}
		} else {
			// if we're not at end of screen then indicate new scanline
			err := tv.newScanline()
			if err != nil {
				return err
			}
		}
	}

	// check vsync signal at the time of the flyback
	if sig.VSync && !tv.state.lastSignal.VSync {
		tv.state.vsyncCount = 0
	} else if sig.VSync && tv.state.lastSignal.VSync {
		tv.state.vsyncCount++
	} else if !sig.VSync && tv.state.lastSignal.VSync {
		if tv.state.vsyncCount > 10 {
			err := tv.newFrame(true)
			if err != nil {
				return err
			}
		}
	}

	// we've "faked" the flyback signal above when clock reached
	// horizClksScanline. we need to handle the real flyback signal however, by
	// making sure we're at the correct clock value.
	//
	// this should be seen as a special condition and one that could be
	// removed if the TV signal was emulated properly. for now the range check
	// is to enable the RSYNC smooth scrolling trick to be displayed correctly.
	//
	// https://atariage.com/forums/topic/224946-smooth-scrolling-playfield-i-think-ive-done-it
	if sig.HSync && !tv.state.lastSignal.HSync {
		if tv.state.clock < 13 || tv.state.clock > 22 {
			tv.state.clock = 16
		}
	}

	// doing nothing with CBURST signal

	// augment television signal before sending to pixel renderer
	sig.Clock = tv.state.clock
	sig.Scanline = tv.state.scanline

	// record the current signal settings so they can be used for reference
	// during the next call to Signal()
	tv.state.lastSignal = sig

	// record signal history
	if tv.currentIdx >= MaxSignalHistory {
		err := tv.processSignals(true)
		if err != nil {
			return err
		}
	}
	tv.signals[tv.currentIdx] = sig
	tv.currentIdx++

	// set pending pixels for pixel-scale frame limiting (but only when the
	// limiter is active - this is important when rendering frames produced
	// durint rewinding)
	if tv.lmtr.limit && tv.lmtr.visualUpdates {
		err := tv.processSignals(true)
		if err != nil {
			return err
		}
	}

	tv.lmtr.measureActual()

	return nil
}

func (tv *Television) newScanline() error {
	// notify renderers of new scanline
	for _, r := range tv.renderers {
		err := r.NewScanline(tv.state.scanline)
		if err != nil {
			return err
		}
	}

	tv.lmtr.checkScanline()

	return nil
}

func (tv *Television) newFrame(synced bool) error {
	// a synced frame is one which was generated from a valid VSYNC/VBLANK sequence
	tv.state.syncedFrame = synced

	// increase or reset stable frame count as required
	if tv.state.stableFrames <= stabilityThreshold {
		if tv.state.syncedFrame {
			tv.state.stableFrames++
		} else {
			tv.state.stableFrames = 0
		}
	}

	// specification change from NTSC to PAL.
	if tv.state.spec.ID == specification.SpecNTSC.ID {
		if tv.state.stableFrames > leadingFrames && tv.state.stableFrames < stabilityThreshold {
			if tv.state.auto && tv.state.scanline > specification.SpecNTSC.ScanlinesTotal+excessScanlinesNTSC {
				_ = tv.SetSpec("PAL")
			}
		}
	}

	// commit any resizing that maybe pending
	err := tv.state.resizer.commit(tv)
	if err != nil {
		return err
	}

	// prepare for next frame
	tv.state.frameNum++
	tv.state.scanline = 0

	// note the current index before processSignals() resets the value
	tv.lastMaxIdx = tv.currentIdx

	// set pending pixels for frame-scale frame limiting or if the frame
	// limiter is inactive
	if !tv.lmtr.limit || tv.lmtr.scale == scaleFrame {
		err = tv.processSignals(true)
		if err != nil {
			return err
		}
	}

	// process all FrameTriggers
	for _, r := range tv.frameTriggers {
		err = r.NewFrame(tv.IsStable())
		if err != nil {
			return err
		}
	}

	// check frame rate. checking even if synced == false.
	tv.lmtr.checkFrame()

	return nil
}

// processSignals forwards pixels in the signalHistory buffer to all pixel renderers.
//
// the "current" argument defines how many pixels to push. if all is true then.
func (tv *Television) processSignals(current bool) error {
	if !tv.pauseRendering {
		for _, r := range tv.renderers {
			r.UpdatingPixels(true)

			err := r.SetPixels(tv.signals[:tv.currentIdx], true)
			if err != nil {
				return curated.Errorf("television: %v", err)
			}

			if !current {
				err = r.SetPixels(tv.signals[tv.currentIdx:tv.lastMaxIdx], false)
				if err != nil {
					return curated.Errorf("television: %v", err)
				}
			}

			// for i := 0; i < tv.currentIdx; i++ {
			// 	sig := tv.signals[i]
			// 	err := r.SetPixel(sig, i < tv.currentIdx)
			// 	if err != nil {
			// 		return curated.Errorf("television: %v", err)
			// 	}
			// }
			// if !current {
			// 	for i := tv.currentIdx + 1; i < tv.lastMaxIdx; i++ {
			// 		sig := tv.signals[i]
			// 		err := r.SetPixel(sig, i < tv.currentIdx)
			// 		if err != nil {
			// 			return curated.Errorf("television: %v", err)
			// 		}
			// 	}
			// }

			r.UpdatingPixels(false)
		}

		// mix audio
		for _, m := range tv.mixers {
			for i := 0; i < tv.currentIdx; i++ {
				sig := tv.signals[i]
				if sig.AudioUpdate {
					err := m.SetAudio(sig.AudioData)
					if err != nil {
						return err
					}
				}
			}
			if !current {
				for i := tv.currentIdx + 1; i < tv.lastMaxIdx; i++ {
					sig := tv.signals[i]
					if sig.AudioUpdate {
						err := m.SetAudio(sig.AudioData)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// reset signal history
	tv.currentIdx = 0

	return nil
}

// IsStable returns true if the television thinks the image being sent by
// the VCS is stable.
func (tv *Television) IsStable() bool {
	return tv.state.stableFrames >= stabilityThreshold
}

// GetLastSignal Returns a copy of the most SignalAttributes sent to the TV
// (via the Signal() function).
func (tv *Television) GetLastSignal() signal.SignalAttributes {
	return tv.state.lastSignal
}

// GetState returns state information for the TV.
func (tv *Television) GetState(request signal.StateReq) int {
	return tv.state.GetState(request)
}

// SetSpecConditional sets the television's specification if the original
// specification (not the current spec, the original) is "AUTO".
//
// This is used when attaching a cartridge to the VCS and also when processing
// setup entries (see setup package, particularly the TV type).
func (tv *Television) SetSpecConditional(spec string) error {
	if tv.GetReqSpecID() == "AUTO" {
		return tv.SetSpec(spec)
	}
	return nil
}

// SetSpec sets the television's specification. Will return an error if
// specification is not recognised.
//
// Currently supported NTSC, PAL, PAL60 and AUTO. The empty string behaves like
// "AUTO".
func (tv *Television) SetSpec(spec string) error {
	switch strings.ToUpper(spec) {
	case "NTSC":
		tv.state.spec = specification.SpecNTSC
		tv.state.auto = false
	case "PAL":
		tv.state.spec = specification.SpecPAL
		tv.state.auto = false
	case "PAL60":
		tv.state.spec = specification.SpecPAL60
		tv.state.auto = false
	case "":
		// the empty string is treated like AUTO
		fallthrough
	case "AUTO":
		tv.state.spec = specification.SpecNTSC
		tv.state.auto = true
	default:
		return curated.Errorf("television: unsupported spec (%s)", spec)
	}

	tv.state.top = tv.state.spec.AtariSafeTop
	tv.state.bottom = tv.state.spec.AtariSafeBottom
	tv.state.resizer.initialise(tv)
	tv.lmtr.setRate(tv.state.spec.FramesPerSecond)

	for _, r := range tv.renderers {
		err := r.Resize(tv.state.spec, tv.state.top, tv.state.bottom)
		if err != nil {
			return err
		}
	}

	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(spec)
	}

	return nil
}

// GetReqSpecID returns the specification that was requested on creation.
func (tv *Television) GetReqSpecID() string {
	return tv.reqSpecID
}

// GetSpec returns the television's current specification. Renderers should use
// GetSpec() rather than keeping a private pointer to the specification.
func (tv *Television) GetSpec() specification.Spec {
	return tv.state.spec
}

// Pause indicates that emulation has been paused. All unpushed pixels will be
// pushed to immeditately. Not the same as PauseRendering(). Pause() should be
// used when emulation is stopped. In this case, paused rendering is implied.
func (tv *Television) Pause(pause bool) error {
	for _, p := range tv.pauseTriggers {
		if err := p.Pause(pause); err != nil {
			return err
		}
	}
	if pause {
		return tv.processSignals(true)
	}
	return nil
}

// PauseRendering halts all forwarding to attached pixel renderers. Not the
// same as Pause(). PauseRendering() should be used when emulation is running
// but no rendering is to take place.
func (tv *Television) PauseRendering(pause bool) {
	tv.pauseRendering = pause
	if !tv.pauseRendering {
		tv.processSignals(false)
	}
}

// SetFPSCap whether the emulation should wait for FPS limiter. Returns the
// setting as it was previously.
func (tv *Television) SetFPSCap(limit bool) bool {
	cap := tv.lmtr.limit
	tv.lmtr.limit = limit
	return cap
}

// SetFPS requests the number frames per second. This overrides the frame rate of
// the specification. A negative  value restores the spec's frame rate.
func (tv *Television) SetFPS(fps float32) {
	tv.lmtr.setRate(fps)
}

// GetReqFPS returns the requested number of frames per second. Compare with
// GetActualFPS() to check for accuracy.
//
// IS goroutine safe.
func (tv *Television) GetReqFPS() float32 {
	return tv.lmtr.requested.Load().(float32)
}

// GetActualFPS returns the current number of frames per second. Note that FPS
// measurement still works even when frame capping is disabled.
//
// IS goroutine safe.
func (tv *Television) GetActualFPS() float32 {
	return tv.lmtr.actual.Load().(float32)
}

// ReqAdjust requests the frame, scanline and clock values where the requested
// StateReq has been adjusted by the specified value. All values will be
// adjusted as required.
//
// The reset argument instructs the function to return values that have been
// reset to zero as appropriate. So when request is ReqFramenum, the scanline
// and clock values will be zero; when request is ReqScanline, the clock value
// will be zero. It has no affect when request is ReqClock.
//
// In the case of a StateAdj of AdjCPUCycle the only allowed adjustment value
// is -1. Any other value will return an error.
func (tv *Television) ReqAdjust(request signal.StateAdj, adjustment int, reset bool) (int, int, int, error) {
	clock := tv.state.clock
	scanline := tv.state.scanline
	frame := tv.state.frameNum

	var err error

	switch request {
	case signal.AdjCPUCycle:
		// adjusting by CPU cycle is the same as adjusting by video cycle
		// accept to say that a CPU cycle is the equivalent of 3 video cycles
		adjustment *= 3
		fallthrough
	case signal.AdjClock:
		clock += adjustment
		if clock >= specification.ClksScanline {
			clock -= specification.ClksScanline
			scanline++
		} else if clock < 0 {
			clock += specification.ClksScanline
			scanline--
		}
		if scanline > tv.state.bottom {
			scanline -= tv.state.bottom
			frame++
		} else if scanline < 0 {
			scanline += tv.state.bottom
			frame--
		}
		if frame < 0 {
			frame = 0
			scanline = 0
			clock = 0
		}
	case signal.AdjInstruction:
		if adjustment != -1 {
			err = curated.Errorf("television: can only adjust CPU boundary by -1")
		} else {
			clock = tv.state.boundaryClock
			scanline = tv.state.boundaryScanline
			frame = tv.state.boundaryFrameNum
		}
	case signal.AdjScanline:
		if reset {
			clock = 0
		}
		scanline += adjustment
		if scanline > tv.state.bottom {
			scanline -= tv.state.bottom
			frame++
		} else if scanline < 0 {
			scanline += tv.state.bottom
			frame--
		}
		if frame < 0 {
			frame = 0
			scanline = 0
		}
	case signal.AdjFramenum:
		if reset {
			clock = 0
			scanline = 0
		}
		frame += adjustment
		if frame < 0 {
			frame = 0
		}
	}

	return frame, scanline, clock - specification.ClksHBlank, err
}

// InstructionBoundary implements the cpu.BoundaryTrigger interface.
func (tv *Television) InstructionBoundary() {
	tv.state.boundaryClock = tv.state.clock
	tv.state.boundaryFrameNum = tv.state.frameNum
	tv.state.boundaryScanline = tv.state.scanline
}
