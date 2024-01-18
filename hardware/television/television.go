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

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
)

// the number of synced frames where we can expect things to be in flux.
const leadingFrames = 1

// the number of synced frames required before the tv is considered to be
// "stable". once the tv is stable then specification switching cannot happen.
//
// resizing can still happen however. so this value is important if we don't
// want to see the screen jump on ROM startup
const stabilityThreshold = 6

// State encapsulates the television values that can change from moment to
// moment. Used by the rewind system when recording the current television
// state.
type State struct {
	// the FrameInfo for the current frame
	frameInfo FrameInfo

	// auto flag indicates that the tv type/specification should switch if it
	// appears to be outside of the current spec.
	//
	// in practice this means that if auto is true then we start with the NTSC
	// spec and move to PAL if the number of scanlines exceeds the NTSC maximum
	auto bool

	// state of the television. these values correspond to the most recent
	// signal received
	//
	// not using TelevisionCoords type here.
	//
	// clock field counts from zero not -specification.ClksHblank
	frameNum int
	scanline int
	clock    int

	// the number of consistent frames seen after reset. once the count reaches
	// stabilityThreshold then Stable flag in the FrameInfo type is set to
	// true.
	//
	// once stableFrames reaches stabilityThreshold it is never reset except by
	// an explicit call to Reset()
	stableFrames int

	// record of signal attributes from the last call to Signal()
	lastSignal signal.SignalAttributes

	// vsync control
	vsyncActive       bool
	vsyncStartOnClock int
	vsyncScanlines    int
	vsyncClocks       int

	// frame resizer
	resizer resizer

	// the coords of the last CPU instruction
	lastCPUInstruction coords.TelevisionCoords
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

// GetLastSignal returns a copy of the most SignalAttributes sent to the TV
// (via the Signal() function).
func (s *State) GetLastSignal() signal.SignalAttributes {
	return s.lastSignal
}

// GetFrameInfo returns the television's current frame information.
func (s *State) GetFrameInfo() FrameInfo {
	return s.frameInfo
}

// GetCoords returns an instance of coords.TelevisionCoords.
func (s *State) GetCoords() coords.TelevisionCoords {
	return coords.TelevisionCoords{
		Frame:    s.frameNum,
		Scanline: s.scanline,
		Clock:    s.clock - specification.ClksHBlank,
	}
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

	// list of ScanlineTrigger implementations to consult
	scanlineTriggers []ScanlineTrigger

	// list of audio mixers to consult
	mixers []AudioMixer

	// realtime mixer. only one allowed
	realtimeMixer RealtimeAudioMixer

	// instance of current state (as supported by the rewind system)
	state *State

	// signals are buffered before being forwarded to a PixelRenderer.
	//
	// signals in the array are always consecutive.
	//
	// the signals in the array will never cross a frame boundary. ie. all
	// signals belong to the same frame.
	//
	// the first signal in the array is not necessary at scanline zero, clock
	// zero
	//
	// information about which scanline/clock a SignalAttribute corresponds to
	// is part of the SignaalAttributes information (see signal package).
	//
	// because each SignalAttribute can be decoded for scanline and clock
	// information the array can be sliced freely
	signals []signal.SignalAttributes

	// the index of the most recent Signal()
	currentSignalIdx int

	// the index of the first Signal() in the frame
	firstSignalIdx int

	// copy of the signals and index fields from the previous frame. we use
	// solely to support the realtime audio mixer
	//
	// updated in renderSignals() function. might need more nuanced
	// copying/appending. for example if renderSignals() is called multiple
	// times per frame. currently this will only happen in the debugger when
	// execution is halted mid frame so I don't think it's an issue
	prevSignals       []signal.SignalAttributes
	prevSignalLastIdx int
	prevSignalFirst   int

	// state of emulation
	emulationState govern.State
}

// NewTelevision creates a new instance of the television type, satisfying the
// Television interface.
func NewTelevision(spec string) (*Television, error) {
	tv := &Television{
		reqSpecID:   strings.ToUpper(spec),
		state:       &State{},
		signals:     make([]signal.SignalAttributes, specification.AbsoluteMaxClks),
		prevSignals: make([]signal.SignalAttributes, specification.AbsoluteMaxClks),
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
	tv.state.vsyncActive = false
	tv.state.vsyncStartOnClock = 0
	tv.state.vsyncScanlines = 0
	tv.state.vsyncClocks = 0
	tv.state.lastSignal = signal.NoSignal

	for i := range tv.signals {
		tv.signals[i] = signal.NoSignal
	}
	tv.currentSignalIdx = 0
	tv.firstSignalIdx = 0

	if tv.state.auto {
		tv.SetSpec("AUTO")
	} else {
		tv.SetSpec(tv.state.frameInfo.Spec.ID)
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
		tv.vcs.SetClockSpeed(tv.state.frameInfo.Spec.ID)
	}

	// reset signal history
	tv.currentSignalIdx = 0
	tv.firstSignalIdx = 0
}

// AttachVCS attaches an implementation of the VCSReturnChannel.
func (tv *Television) AttachVCS(vcs VCSReturnChannel) {
	tv.vcs = vcs

	// notify the newly attached console of the current TV spec
	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(tv.state.frameInfo.Spec.ID)
	}
}

// AddPixelRenderer adds an implementation of PixelRenderer.
func (tv *Television) AddPixelRenderer(r PixelRenderer) {
	for i := range tv.renderers {
		if tv.renderers[i] == r {
			return
		}
	}
	tv.renderers = append(tv.renderers, r)
}

// RemovePixelRenderer removes a single PixelRenderer implementation from the
// list of renderers. Order is not maintained.
func (tv *Television) RemovePixelRenderer(r PixelRenderer) {
	for i := range tv.renderers {
		if tv.renderers[i] == r {
			tv.renderers[i] = tv.renderers[len(tv.renderers)-1]
			tv.renderers = tv.renderers[:len(tv.renderers)-1]
			return
		}
	}
}

// AddFrameTrigger adds an implementation of FrameTrigger.
func (tv *Television) AddFrameTrigger(f FrameTrigger) {
	for i := range tv.frameTriggers {
		if tv.frameTriggers[i] == f {
			return
		}
	}
	tv.frameTriggers = append(tv.frameTriggers, f)
}

// RemoveFrameTrigger removes a single FrameTrigger implementation from the
// list of triggers. Order is not maintained.
func (tv *Television) RemoveFrameTrigger(f FrameTrigger) {
	for i := range tv.frameTriggers {
		if tv.frameTriggers[i] == f {
			tv.frameTriggers[i] = tv.frameTriggers[len(tv.frameTriggers)-1]
			tv.frameTriggers = tv.frameTriggers[:len(tv.frameTriggers)-1]
			return
		}
	}
}

// AddScanlineTrigger adds an implementation of ScanlineTrigger.
func (tv *Television) AddScanlineTrigger(f ScanlineTrigger) {
	for i := range tv.scanlineTriggers {
		if tv.scanlineTriggers[i] == f {
			return
		}
	}
	tv.scanlineTriggers = append(tv.scanlineTriggers, f)
}

// RemoveScanlineTrigger removes a single ScanlineTrigger implementation from the
// list of triggers. Order is not maintained.
func (tv *Television) RemoveScanlineTrigger(f ScanlineTrigger) {
	for i := range tv.scanlineTriggers {
		if tv.scanlineTriggers[i] == f {
			tv.scanlineTriggers[i] = tv.scanlineTriggers[len(tv.scanlineTriggers)-1]
			tv.scanlineTriggers = tv.scanlineTriggers[:len(tv.scanlineTriggers)-1]
			return
		}
	}
}

// AddAudioMixer adds an implementation of AudioMixer.
func (tv *Television) AddAudioMixer(m AudioMixer) {
	for i := range tv.mixers {
		if tv.mixers[i] == m {
			return
		}
	}
	tv.mixers = append(tv.mixers, m)
}

// RemoveAudioMixer removes a single AudioMixer implementation from the
// list of mixers. Order is not maintained.
func (tv *Television) RemoveAudioMixer(m AudioMixer) {
	for i := range tv.mixers {
		if tv.mixers[i] == m {
			tv.mixers[i] = tv.mixers[len(tv.mixers)-1]
			tv.mixers = tv.mixers[:len(tv.mixers)-1]
			return
		}
	}
}

// AddRealtimeAudioMixer adds a RealtimeAudioMixer. Any previous assignment is
// lost.
func (tv *Television) AddRealtimeAudioMixer(m RealtimeAudioMixer) {
	tv.realtimeMixer = m
}

// RemoveRealtimeAudioMixer removes a RealtimeAudioMixer implementation from
// the Television.
func (tv *Television) RemoveRealtimeAudioMixer(m RealtimeAudioMixer) {
	tv.realtimeMixer = m
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
func (tv *Television) Signal(sig signal.SignalAttributes) {
	// examine signal for resizing possibility.
	//
	// throttle how often we do this because it's an expensive operation. the
	// range check is required because the decision to commit a resize takes
	// several frames (defined by framesUntilResize)
	//
	// if the frame is not stable then we always perfom the check. this is so
	// we don't see a resizing frame too often because a resize is likely
	// during startup - Spike's Peak is a good example of a resizing ROM
	//
	// the throttle does mean there can be a slight delay before a resize is
	// committed but this is rare (Hack'Em Hanglyman is a good example of such
	// a ROM) and the performance benefits are significant
	if !tv.state.frameInfo.Stable || tv.state.frameNum%16 <= framesUntilResize {
		tv.state.resizer.examine(tv, sig)
	}

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

		// fly-back naturally if VBlank is off. a good example of a ROM
		// that requires correct handling of this is Andrew Davies' Chess
		// (3e+ test rom)
		//
		// (06/01/21) another example is the Artkaris NTSC version of Lili
		if tv.state.scanline >= specification.AbsoluteMaxScanlines {
			// (20/10/22) I'm no longer sure if this test for an active vsync
			// is necessary. it might be better / more accurate if the test is
			// removed and the screen allowed to flyback regardless of the
			// VSYNC state
			//
			// however, without testing I'm no longer sure what the effect of
			// that be. in particular how the results appear in the debugging
			// screen and specically, how it affects the debugging screen's
			// onion skinning
			//
			// I'll leave it in place for now until further testing can be done
			if !tv.state.vsyncActive {
				err := tv.newFrame(false)
				if err != nil {
					logger.Log("TV", err.Error())
				}
			} else {
				tv.state.scanline = specification.AbsoluteMaxScanlines - 1
			}
		} else {
			// if we're not at end of screen then indicate new scanline
			err := tv.newScanline()
			if err != nil {
				logger.Log("TV", err.Error())
			}
		}
	}

	// count VSYNC clocks and scanlines
	if tv.state.vsyncActive {
		if tv.state.clock == tv.state.vsyncStartOnClock {
			tv.state.vsyncScanlines++
		}
		tv.state.vsyncClocks++
	}

	// check for change of VSYNC signal
	if sig&signal.VSync != tv.state.lastSignal&signal.VSync {
		if sig&signal.VSync == signal.VSync {
			// VSYNC has started
			tv.state.vsyncActive = true
			tv.state.vsyncScanlines = 0
			tv.state.vsyncStartOnClock = tv.state.clock
		} else {
			// VSYNC has ended but we don't want to trigger a new frame unless
			// the VSYNC signal has been present for a minimum number of
			// clocks
			//
			// there's no real empirical reason for the value used here except
			// that it seems right in practice. it certainly doesn't seem to
			// cause any harm.
			//
			// it's worth noting that without this minimum threshold the
			// smoothscrolling demos (mentioned below) don't work as expected.
			// so maybe there's a subtle interaction with RSYNC here that's
			// worth exploring
			if tv.state.vsyncClocks > 10 {
				err := tv.newFrame(true)
				if err != nil {
					logger.Log("TV", err.Error())
				}
			}
			tv.state.vsyncActive = false
		}
		tv.state.vsyncClocks = 0
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
	if sig&signal.HSync == signal.HSync && tv.state.lastSignal&signal.HSync != signal.HSync {
		if tv.state.clock < 13 || tv.state.clock > 22 {
			tv.state.clock = 16

		}
	}

	// doing nothing with CBURST signal

	// assume that clock and scanline are constrained elsewhere such that the
	// index can never run past the end of the signals array
	tv.currentSignalIdx = tv.state.clock + (tv.state.scanline * specification.ClksScanline)

	// sometimes the current signal can come out "behind" the firstSignalIdx.
	// this can happen when RSYNC is triggered on the first scanline of the
	// frame. not common but we should handle it
	//
	// in practical terms, if we don't handle this then sending signals to the
	// audio mixers will cause a "slice bounds out of range" panic
	if tv.currentSignalIdx < tv.firstSignalIdx {
		tv.firstSignalIdx = tv.currentSignalIdx
	}

	// augment television signal before storing and sending to pixel renderers
	sig &= ^signal.Index
	sig |= signal.SignalAttributes(tv.currentSignalIdx << signal.IndexShift)

	// write the signal into the correct index of the signals array.

	tv.signals[tv.currentSignalIdx] = sig

	// record the current signal settings so they can be used for reference
	// during the next call to Signal()
	tv.state.lastSignal = sig

	// record signal history
	if tv.currentSignalIdx >= len(tv.signals) {
		err := tv.renderSignals()
		if err != nil {
			logger.Log("TV", err.Error())
		}
	}
}

func (tv *Television) newScanline() error {
	// notify renderers of new scanline
	for _, r := range tv.renderers {
		err := r.NewScanline(tv.state.scanline)
		if err != nil {
			return err
		}
	}

	// check for realtime mixing requirements. if it is required then
	// immediately push the audio data from the previous frame to the mixer
	if tv.realtimeMixer != nil && tv.emulationState == govern.Running && tv.state.frameInfo.Stable {
		if tv.realtimeMixer.MoreAudio() {
			err := tv.realtimeMixer.SetAudio(tv.prevSignals[:tv.prevSignalLastIdx])
			if err != nil {
				return err
			}
		}
	}

	// process all ScanlineTriggers
	for _, r := range tv.scanlineTriggers {
		err := r.NewScanline(tv.state.frameInfo)
		if err != nil {
			return err
		}
	}

	tv.lmtr.checkScanline()

	return nil
}

// the fromVsync arguments is true if a valid VSYNC signal has been detected. a
// value of false means the frame flyback is unsynced.
func (tv *Television) newFrame(fromVsync bool) error {
	// increase or reset stable frame count as required
	if tv.state.stableFrames <= stabilityThreshold {
		if fromVsync {
			tv.state.stableFrames++
			tv.state.frameInfo.Stable = tv.state.stableFrames >= stabilityThreshold
		} else {
			tv.state.stableFrames = 0
			tv.state.frameInfo.Stable = false
		}
	}

	// specification change between NTSC and PAL. PAL60 is treated the same as
	// NTSC in this instance
	//
	// Note that SetSpec() resets the frameInfo completely so we must set the
	// framenumber and vsynced after any possible SetSpec()
	if tv.state.stableFrames > leadingFrames && tv.state.stableFrames < stabilityThreshold {
		switch tv.state.frameInfo.Spec.ID {
		case specification.SpecPALM.ID:
			fallthrough
		case specification.SpecNTSC.ID:
			if tv.state.auto && tv.state.scanline > specification.PALTrigger {
				_ = tv.SetSpec("PAL")
			}
		case specification.SpecPAL.ID:
			if tv.state.auto && tv.state.scanline <= specification.PALTrigger {
				_ = tv.SetSpec("NTSC")
			}
		}
	}

	// update frame number
	tv.state.frameInfo.FrameNum = tv.state.frameNum

	// note whether newFrame() was the result of a valid VSYNC or a "natural" flyback
	tv.state.frameInfo.VSync = fromVsync
	tv.state.frameInfo.VSyncScanlines = tv.state.vsyncScanlines

	// commit any resizing that maybe pending
	err := tv.state.resizer.commit(tv)
	if err != nil {
		return err
	}

	// record total scanlines and refresh rate if changed. note that this is
	// independent of the resizer.commit() call above. total scanline / refresh
	// rate can change without it being a resize
	//
	// this is important to do and failure to set the refresh reate correctly
	// is most noticeable in the Supercharger tape loading process. During tape
	// loading a steady sine wave is produced and no VSYNC is issued. This
	// means that the refresh rate is reduced to 50.27Hz
	//
	// the disadvantage of disassociating screen size (by which we mean the
	// period between VSYNCs) from the refresh rate is that debugging
	// information may be misleading. but that's really not a problem we should
	// be directly addressing in the television package
	if tv.state.frameInfo.TotalScanlines != tv.state.scanline {
		tv.state.frameInfo.TotalScanlines = tv.state.scanline
		tv.state.frameInfo.RefreshRate = 15734.26 / float32(tv.state.scanline)
		tv.lmtr.setRefreshRate(tv.state.frameInfo.RefreshRate)
		tv.state.frameInfo.Jitter = true
	} else {
		tv.state.frameInfo.Jitter = false
	}

	// prepare for next frame
	tv.state.frameNum++
	tv.state.scanline = 0

	// nullify unused signals at end of frame
	for i := tv.currentSignalIdx; i < len(tv.signals); i++ {
		tv.signals[i] = signal.NoSignal
	}

	// set pending pixels
	err = tv.renderSignals()
	if err != nil {
		return err
	}

	// process all pixel renderers
	for _, r := range tv.renderers {
		err := r.NewFrame(tv.state.frameInfo)
		if err != nil {
			return err
		}
	}

	// process all FrameTriggers
	for _, r := range tv.frameTriggers {
		err := r.NewFrame(tv.state.frameInfo)
		if err != nil {
			return err
		}
	}

	// check frame rate
	tv.lmtr.checkFrame()

	// measure frame rate
	tv.lmtr.measureActual()

	// signal index at beginning of new frame
	tv.firstSignalIdx = tv.state.clock + (tv.state.scanline * specification.ClksScanline)

	return nil
}

// renderSignals forwards pixels in the signalHistory buffer to all pixel
// renderers and audio mixers.
func (tv *Television) renderSignals() error {
	// do not render pixels if emulation is in the rewinding state
	if tv.emulationState != govern.Rewinding {
		for _, r := range tv.renderers {
			err := r.SetPixels(tv.signals, tv.currentSignalIdx)
			if err != nil {
				return fmt.Errorf("television: %w", err)
			}
		}
	}

	// update realtime mixers
	//
	// an additional condition saying the realtimeMixer is used only once the
	// frame is stable has been removed. it was thought to improve sound on
	// startup for some ROMs but in some pathological cases it means sound is
	// never output. in particular, the tunabit demo ROM.
	//
	// https://atariage.com/forums/topic/274172-tiatune-tia-music-player-with-correct-tuning/
	if tv.realtimeMixer != nil {
		err := tv.realtimeMixer.SetAudio(tv.signals[tv.firstSignalIdx:tv.currentSignalIdx])
		if err != nil {
			return fmt.Errorf("television: %w", err)
		}
	}

	// but we do mix audio even if the emulation is rewinding
	for _, m := range tv.mixers {
		err := m.SetAudio(tv.signals[tv.firstSignalIdx:tv.currentSignalIdx])
		if err != nil {
			return fmt.Errorf("television: %w", err)
		}
	}

	// make a copy of signals just rendered
	copy(tv.prevSignals, tv.signals)
	tv.prevSignalLastIdx = tv.currentSignalIdx
	tv.prevSignalFirst = tv.firstSignalIdx

	return nil
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
		tv.state.frameInfo = NewFrameInfo(specification.SpecNTSC)
		tv.state.auto = false
	case "PAL":
		tv.state.frameInfo = NewFrameInfo(specification.SpecPAL)
		tv.state.auto = false
	case "PAL-M":
		tv.state.frameInfo = NewFrameInfo(specification.SpecPALM)
		tv.state.auto = false
	case "SECAM":
		tv.state.frameInfo = NewFrameInfo(specification.SpecSECAM)
		tv.state.auto = false
	case "":
		// the empty string is treated like AUTO
		fallthrough
	case "AUTO":
		tv.state.frameInfo = NewFrameInfo(specification.SpecNTSC)
		tv.state.auto = true
	default:
		return fmt.Errorf("television: unsupported spec (%s)", spec)
	}

	tv.state.resizer.initialise(tv)
	tv.lmtr.setRefreshRate(tv.state.frameInfo.Spec.RefreshRate)
	tv.lmtr.setRate(tv.state.frameInfo.Spec.RefreshRate)

	for _, r := range tv.renderers {
		r.Reset()
	}

	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(tv.state.frameInfo.Spec.ID)
	}

	return nil
}

// SetEmulationState is called by emulation whenever state changes. How we
// handle incoming signals depends on the current state.
func (tv *Television) SetEmulationState(state govern.State) error {
	prev := tv.emulationState
	tv.emulationState = state

	switch prev {
	case govern.Paused:
		// start off the unpaused state by measuring the current framerate.
		// this "clears" the ticker channel and means the feedback from
		// GetActualFPS() is less misleading
		tv.lmtr.measureActual()

	case govern.Rewinding:
		tv.renderSignals()
	}

	switch state {
	case govern.Paused:
		err := tv.renderSignals()
		if err != nil {
			return err
		}
	}

	return nil
}

// NudgeFPSCap stops the FPS limiter for the specified number of frames. A value
// of zero (or less) will stop any existing nudge
func (tv *Television) NudgeFPSCap(frames int) {
	if frames < 0 {
		frames = 0
	}
	tv.lmtr.nudge.Store(int32(frames))
}

// SetFPSCap whether the emulation should wait for FPS limiter. Returns the
// setting as it was previously.
func (tv *Television) SetFPSCap(limit bool) bool {
	prev := tv.lmtr.active
	tv.lmtr.active = limit

	// notify all pixel renderers that are interested in the FPS cap
	for i := range tv.renderers {
		if r, ok := tv.renderers[i].(PixelRendererFPSCap); ok {
			r.SetFPSCap(limit)
		}
	}

	return prev
}

// SetFPS requests the number frames per second. This overrides the frame rate of
// the specification. A negative value restores frame rate to the ideal value
// (the frequency of the incoming signal).
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

// GetActualFPS returns the current number of frames per second and the
// detected frequency of the TV signal.
//
// Note that FPS measurement still works even when frame capping is disabled.
//
// IS goroutine safe.
func (tv *Television) GetActualFPS() (float32, float32) {
	return tv.lmtr.measured.Load().(float32), tv.lmtr.refreshRate.Load().(float32)
}

// GetReqSpecID returns the specification that was requested on creation.
func (tv *Television) GetReqSpecID() string {
	return tv.reqSpecID
}

// GetSpecID returns the current specification.
func (tv *Television) GetSpecID() string {
	return tv.state.frameInfo.Spec.ID
}

// GetFrameInfo returns the television's current frame information.
func (tv *Television) GetFrameInfo() FrameInfo {
	return tv.state.frameInfo
}

// GetLastSignal returns a copy of the most SignalAttributes sent to the TV
// (via the Signal() function).
func (tv *Television) GetLastSignal() signal.SignalAttributes {
	return tv.state.lastSignal
}

// GetCoords returns an instance of coords.TelevisionCoords.
//
// Like all Television functions this function is not safe to call from
// goroutines other than the one that created the Television.
func (tv *Television) GetCoords() coords.TelevisionCoords {
	return tv.state.GetCoords()
}

// SetRotation instructs the television to a different orientation. In truth,
// the television just forwards the request to the pixel renderers.
func (tv *Television) SetRotation(rotation specification.Rotation) {
	for _, r := range tv.renderers {
		if s, ok := r.(PixelRendererRotation); ok {
			s.SetRotation(rotation)
		}
	}
}
