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

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/limiter"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
)

// the number of synced frames where we can expect things to be in flux.
const leadingFrames = 1

// the number of synced frames required before the TV is considered to be
// stable. once the tv is stable then specification switching cannot happen.
const stabilityThreshold = 6

// State encapsulates the television values that can change from moment to
// moment. Used by the rewind system when recording the current television
// state.
type State struct {
	// the FrameInfo for the current frame
	frameInfo frameinfo.Current

	// the specification to use when the television is reset
	resetSpec string

	// whether the current specification was decided upon by an AUTO request
	autoSpec bool

	// state of the television. these values correspond to the most recent
	// signal received
	//
	// not using TelevisionCoords type here.
	//
	// clock field counts from zero not negative specification.ClksHblank
	frameNum int
	scanline int
	clock    int

	// the number of VSYNCED frames seen after reset. once the count reaches
	// stabilityThreshold then Stable flag in the FrameInfo type is set to
	// true.
	//
	// once stableFrames reaches stabilityThreshold it is never reset except by
	// an explicit call to Reset() or by SetSpec() with the forced flag and a
	// requested spec of "AUTO"
	stableFrames int

	// record of signal attributes from the last call to Signal()
	lastSignal signal.SignalAttributes

	// vsync control
	vsync vsync

	// latch to say if next flyback was a result of VSYNC or not
	fromVSYNC bool

	// if VSYNC was attempted but failed due to the current TV settings. in that
	// case fromVSYNC will be false and failedVSYNC will be true
	//
	// we use this to prevent the setRefreshRate() function from changing the
	// audio refresh rate if the ROM is at least trying to synchronise
	failedVSYNC bool

	// frame resizer
	resizer Resizer

	// bounds detection for VBLANK signal. similar to but not the same as the
	// resizer
	bounds vblankBounds
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

// spec string MUST be normalised with specification.NormaliseReqSpecID()
func (s *State) setSpec(spec string) {
	spec, ok := specification.NormaliseReqSpecID(spec)
	if !ok {
		return
	}

	// conert AUTO to the resetSpec value
	if spec == "AUTO" {
		spec = s.resetSpec
	}

	// if spec is still AUTO then we default to NTSC

	switch spec {
	case "AUTO", "NTSC":
		s.frameInfo = frameinfo.NewCurrent(specification.SpecNTSC)
		s.resizer.reset(specification.SpecNTSC)
		s.autoSpec = spec == "AUTO"
	case "PAL":
		s.frameInfo = frameinfo.NewCurrent(specification.SpecPAL)
		s.resizer.reset(specification.SpecPAL)
		s.autoSpec = false
	case "PAL60":
		s.frameInfo = frameinfo.NewCurrent(specification.SpecPAL60)
		s.resizer.reset(specification.SpecPAL60)
		s.autoSpec = false
	case "PAL-M":
		s.frameInfo = frameinfo.NewCurrent(specification.SpecPAL_M)
		s.resizer.reset(specification.SpecPAL_M)
		s.autoSpec = false
	case "SECAM":
		s.frameInfo = frameinfo.NewCurrent(specification.SpecSECAM)
		s.resizer.reset(specification.SpecSECAM)
		s.autoSpec = false
	}

	s.stableFrames = 0
}

// SetSpec sets the requested specification ID
func (s *State) SetSpec(spec string) {
	s.setSpec(spec)
}

// GetLastSignal returns a copy of the most SignalAttributes sent to the TV
// (via the Signal() function).
func (s *State) GetLastSignal() signal.SignalAttributes {
	return s.lastSignal
}

// GetFrameInfo returns the television's current frame information.
func (s *State) GetFrameInfo() frameinfo.Current {
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
	env *environment.Environment

	// vcs will be nil unless AttachVCS() has been called
	vcs VCS

	// interface to a debugger
	debugger Debugger

	// framerate limiter
	lmtr *limiter.Limiter

	// list of PixelRenderer implementations to consult
	renderers []PixelRenderer

	// the most recently added pixel rendererer that implements PixelRendererDisplay. there can only
	// be one of these at any one time and a new one added will replace the previous display
	rendererDisplay PixelRendererDisplay

	// list of FrameTrigger implementations to consult
	frameTriggers []FrameTrigger

	// list of ScanlineTrigger implementations to consult
	scanlineTriggers []ScanlineTrigger

	// list of audio mixers to consult
	mixers []AudioMixer

	realTimeMixer RealtimeAudioMixer

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
	//
	// currentSignalIdx is the index of the most recent Signal()
	//
	// firstSignalIdx the index of the first Signal() in the frame
	signals          []signal.SignalAttributes
	currentSignalIdx int
	firstSignalIdx   int

	audioSignals     []signal.AudioSignalAttributes
	audioSignalLimit int

	// state of emulation
	emulationState govern.State
}

const baselineAudioSignalLimit = 200

// NewTelevision creates a new instance of the television type, satisfying the
// Television interface.
func NewTelevision(spec string) (*Television, error) {
	spec, ok := specification.NormaliseReqSpecID(spec)
	if !ok {
		return nil, fmt.Errorf("television: unsupported spec (%s)", spec)
	}

	tv := &Television{
		signals: make([]signal.SignalAttributes, specification.AbsoluteMaxClks),
		state: &State{
			resetSpec: spec,
		},
	}

	// initialise frame rate limiter
	tv.lmtr = limiter.NewLimiter()
	tv.SetFPS(limiter.MatchRefreshRate)

	// empty list of renderers
	tv.renderers = make([]PixelRenderer, 0)

	// all other intialisation happens as part of a reset opertion
	tv.Reset(false)

	return tv, nil
}

func (tv *Television) String() string {
	return tv.state.String()
}

func (tv *Television) SpecString() string {
	return fmt.Sprintf("current=%s reset=%s", tv.state.frameInfo.Spec.ID, tv.state.resetSpec)
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

	tv.setSpec(tv.state.resetSpec)

	tv.state.clock = 0
	tv.state.scanline = 0
	tv.state.stableFrames = 0
	tv.state.vsync.reset()
	tv.state.fromVSYNC = false
	tv.state.failedVSYNC = false
	tv.state.lastSignal = signal.SignalAttributes{
		Index: signal.NoSignal,
	}

	for i := range tv.signals {
		tv.signals[i] = signal.SignalAttributes{
			Index: signal.NoSignal,
		}
	}
	tv.currentSignalIdx = 0
	tv.firstSignalIdx = 0

	tv.setRefreshRate(tv.state.frameInfo.Spec.RefreshRate)
	tv.state.resizer.reset(tv.state.frameInfo.Spec)
	tv.state.bounds.reset()

	for _, r := range tv.renderers {
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

// Plumb attaches an existing television state.
func (tv *Television) Plumb(vcs VCS, state *State) {
	if state == nil {
		panic("television: cannot plumb in a nil state")
	}

	tv.state = state.Snapshot()

	// make sure vcs knows about current spec
	tv.vcs = vcs
	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(tv.state.frameInfo.Spec)
	}

	// reset signal history
	tv.currentSignalIdx = 0
	tv.firstSignalIdx = 0
}

// AttachVCS attaches an implementation of the VCSReturnChannel.
func (tv *Television) AttachVCS(env *environment.Environment, vcs VCS) {
	tv.env = env
	tv.vcs = vcs

	// notify the newly attached console of the current TV spec
	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(tv.state.frameInfo.Spec)
	}
}

// AddDebugger adds an implementation of Debugger.
func (tv *Television) AddDebugger(dbg Debugger) {
	tv.debugger = dbg
}

// AddPixelRenderer adds an implementation of PixelRenderer. If the PixelRenderer is also an
// implementation of PixelRendererDisplay then note that there can only be one such renderer and any
// such renderer previously added will be replaced.
func (tv *Television) AddPixelRenderer(r PixelRenderer) {
	if r, ok := r.(PixelRendererDisplay); ok {
		tv.RemovePixelRenderer(tv.rendererDisplay)
		tv.rendererDisplay = r
		tv.lmtr.SetDisplay(r)
	}
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
	if r, ok := r.(PixelRendererDisplay); ok {
		if tv.rendererDisplay == r {
			tv.rendererDisplay = nil
			tv.lmtr.SetDisplay(nil)
		}
	}
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

// SetRealTimeAudioMixer specified which realtime audio mixer to use. Only one
// realtime mixer can be set at once. Unset with nil
func (tv *Television) SetRealTimeAudioMixer(m RealtimeAudioMixer) {
	tv.realTimeMixer = m
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

// AudioSignal updates the audio stream.
func (tv *Television) AudioSignal(sig signal.AudioSignalAttributes) {
	tv.audioSignals = append(tv.audioSignals, sig)
	if len(tv.audioSignals) >= tv.state.frameInfo.TotalScanlines {
		if tv.realTimeMixer != nil {
			err := tv.realTimeMixer.SetAudio(tv.audioSignals[:])
			if err != nil {
				logger.Log(tv.env, "TV", err)
			}
		}

		// update normal mixers
		for _, m := range tv.mixers {
			err := m.SetAudio(tv.audioSignals[:])
			if err != nil {
				logger.Log(tv.env, "TV", err)
			}
		}

		// flush audio signal after both realtime and normal mixers have been processed
		tv.audioSignals = tv.audioSignals[:0]
	}
}

// Signal updates the current state of the television.
func (tv *Television) Signal(sig signal.SignalAttributes) {
	// count VSYNC scanlines based on when HSync ends. this means that if
	// VSYNC is set after HSYNC on a scanline, the scanlines won't be
	// counted for almost an entire scanline
	if tv.state.vsync.active && !sig.HSync && tv.state.lastSignal.HSync {
		tv.state.vsync.activeScanlineCount++
	}

	// check for change of VSYNC signal
	if sig.VSync != tv.state.lastSignal.VSync {
		if sig.VSync {
			// VSYNC has started
			tv.state.vsync.active = true
			tv.state.vsync.activeScanlineCount = 0
			tv.state.vsync.startScanline = tv.state.scanline
			tv.state.vsync.startClock = tv.state.clock

			// check that VSYNC start scanline hasn't changed
			if tv.state.frameInfo.Stable && tv.debugger != nil {
				if tv.state.frameInfo.VSYNCscanline != tv.state.vsync.startScanline {
					if tv.env.Prefs.TV.HaltChangedVSYNC.Get().(bool) {
						tv.debugger.HaltFromTelevision("change of VSYNC start scanline")
					}
					tv.state.frameInfo.VSYNCunstable = true
				} else if tv.state.frameInfo.VSYNCclock != tv.state.vsync.startClock {
					if tv.env.Prefs.TV.HaltChangedVSYNC.Get().(bool) {
						tv.debugger.HaltFromTelevision("change of VSYNC start clock")
					}
					tv.state.frameInfo.VSYNCunstable = true
				}
			}
		} else {
			// check that VSYNC count hasn't changed
			if tv.state.frameInfo.Stable && tv.debugger != nil {
				if tv.state.frameInfo.VSYNCcount != tv.state.vsync.activeScanlineCount {
					if tv.env.Prefs.TV.HaltChangedVSYNC.Get().(bool) {
						tv.debugger.HaltFromTelevision("change of VSYNC count")
					}
					tv.state.frameInfo.VSYNCunstable = true
				}
			}

			// VSYNC has been disabled this cycle
			tv.state.vsync.active = false
			tv.state.fromVSYNC = tv.state.vsync.activeScanlineCount >= tv.env.Prefs.TV.VSYNCscanlines.Get().(int)
			if !tv.state.fromVSYNC {
				// VSYNC was too short
				if tv.state.frameInfo.Stable && tv.debugger != nil {
					if tv.env.Prefs.TV.HaltChangedVSYNC.Get().(bool) {
						tv.debugger.HaltFromTelevision("VSYNC too short")
					}
					tv.state.frameInfo.VSYNCunstable = true
				}
			} else {
				tv.state.failedVSYNC = true
			}
		}
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

		// whether to call newFrame() or newScanline()
		var newFrame bool

		// a very good example of a ROM that requires correct handling of
		// natural flyback is Andrew Davies' Chess (3e+ test rom) and also the
		// supercharger loading screen. the latter has sound and so is a good
		// test for how the television interacts with the audio mixers under
		// unsynchronised conditions
		if tv.state.fromVSYNC {
			newFrame = true

		} else {
			// treat newframe differently depending on whether the previous frame was synced or not
			if tv.state.frameInfo.FromVSYNC {
				if tv.state.scanline >= specification.AbsoluteMaxScanlines {
					newFrame = true

					// new frame is a result of a natural flyback
					if tv.state.frameInfo.Stable && tv.debugger != nil {
						if tv.env.Prefs.TV.HaltChangedVSYNC.Get().(bool) {
							tv.debugger.HaltFromTelevision("no VSYNC (natural flyback)")
						}
						tv.state.frameInfo.VSYNCunstable = true
					}
				}
			} else {
				// this is a continuing unsyncrhonised state over multiple
				// frames. if the scanline goes past the current flyback point,
				// denoted by TotalScanlines, then we trigger a new frame
				newFrame = tv.state.scanline > min(specification.AbsoluteMaxScanlines-1, tv.state.frameInfo.TotalScanlines)
			}
		}

		// call newFrame() or newScanline()
		if newFrame {
			err := tv.newFrame()
			if err != nil {
				logger.Log(tv.env, "TV", err)
			}
		} else {
			err := tv.newScanline()
			if err != nil {
				logger.Log(tv.env, "TV", err)
			}
		}
	}

	// limit resizing/bounds check to one every 4 clocks. this is enough to
	// catch VBLANK changes and any black-pixels as required
	const resizeCheckPeriod = 4

	// a value of 4 also means that the checks will be made at the start of a
	// new scanline and at the end of a new scanline. this is deliberate but we
	// may do better if we exclude the end-of-scanline check. a value of 8 or
	// maybe 16 would achieve this (or keep using a period of 4 and only perform
	// the period test if the clock is not ClksScanline )
	if tv.state.clock%resizeCheckPeriod == 0 {
		// examine signal for resizing possibility.
		tv.state.resizer.examine(tv.state, sig)

		// check VBLANK bounds
		tv.state.bounds.examine(sig, tv.state.scanline)
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

	// assume that clock and scanline are constrained elsewhere such that the
	// index can never run past the end of the signals array
	tv.currentSignalIdx = tv.state.clock + (tv.state.scanline * specification.ClksScanline)

	// currentSignalIdx should be inside the range of the signals array. we
	// don't need to check and we don't need to push the outstanding signals to
	// the pixel renderers
	//
	// the size of the signals array is based on the specification values and
	// the range of the clock and scanline fields are bounded by that
	//
	// if something has gone wrong then the program will panic on it's own. we
	// don't need to add an additional check where the only course of action is
	// to panic

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
	sig.Index = tv.currentSignalIdx

	// write the signal into the correct index of the signals array.
	tv.signals[tv.currentSignalIdx] = sig

	// record the current signal settings so they can be used for reference
	// during the next call to Signal()
	tv.state.lastSignal = sig
}

func (tv *Television) newScanline() error {
	// notify renderers of new scanline
	for _, r := range tv.renderers {
		err := r.NewScanline(tv.state.scanline)
		if err != nil {
			return err
		}
	}

	// process all ScanlineTriggers
	for _, r := range tv.scanlineTriggers {
		err := r.NewScanline(tv.state.frameInfo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (tv *Television) newFrame() error {
	// increase or reset stable frame count as required
	if tv.state.stableFrames <= stabilityThreshold {
		if tv.state.frameInfo.IsSynced() {
			tv.state.stableFrames++
		}
	} else {
		tv.state.frameInfo.Stable = true
	}

	// specification change between NTSC and PAL. PAL-M is treated the same as
	// NTSC in this instance
	//
	// Note that setSpec() resets the frameInfo completely so we must set the
	// framenumber and vsynced after any possible setSpec()
	if tv.state.stableFrames > leadingFrames && tv.state.stableFrames < stabilityThreshold {
		switch tv.state.frameInfo.Spec.ID {
		case specification.SpecPAL_M.ID:
			fallthrough
		case specification.SpecNTSC.ID:
			if tv.state.frameInfo.Spec.ID != "PAL" {
				if tv.state.autoSpec && tv.state.scanline > specification.PALTrigger {
					tv.setSpec("PAL")
				}
			}
		case specification.SpecPAL.ID:
			if tv.state.frameInfo.Spec.ID != "NTSC" {
				if tv.state.autoSpec && tv.state.scanline <= specification.PALTrigger {
					tv.setSpec("NTSC")
				}
			}
		}
	}

	// update frame number
	tv.state.frameInfo.FrameNum = tv.state.frameNum

	// check VBLANK halt condition
	if tv.state.bounds.commit(tv.state) {
		if tv.debugger != nil {
			if tv.env.Prefs.TV.HaltChangedVBLANK.Get().(bool) {
				tv.debugger.HaltFromTelevision("change of VBLANK boundaries")
			}
			tv.state.frameInfo.VBLANKunstable = true
		}
	}

	// commit any resizing that maybe pending
	err := tv.state.resizer.commit(tv.state)
	if err != nil {
		return err
	}

	// values for screen rolling
	const (
		desyncSpeed   = 1.10
		recoverySpeed = 0.80
	)

	if tv.state.frameInfo.TotalScanlines != tv.state.scanline {
		if tv.state.fromVSYNC {
			tv.state.frameInfo.TotalScanlines = tv.state.scanline
		} else {
			// how bad VSYNC
			if tv.env.Prefs.TV.VSYNCimmedateSync.Get().(bool) {
				// size of frame immediately goes to the maximum
				tv.state.frameInfo.TotalScanlines = specification.AbsoluteMaxScanlines

				// reset 'top scanline' value in case the 'immediate sync'
				// option has been enabled sometime after desynchronisation
				// started
				tv.state.vsync.topScanline = 0
			} else {
				// size of frame slowly grows to the maximum
				if tv.state.frameInfo.TotalScanlines < specification.AbsoluteMaxScanlines {
					// +1 so that low scanline values always grow. very low desyncSpeed values
					// might result in a static scanline value
					//
					// An example of a ROM that might not is "Bezerk Voice Enhanced" which creates
					// a very short frame of 44 scanlines before allowing the screen to sync freely
					tv.state.frameInfo.TotalScanlines = int(float64(tv.state.frameInfo.TotalScanlines)*desyncSpeed) + 1
					tv.state.frameInfo.TotalScanlines = min(tv.state.frameInfo.TotalScanlines, specification.AbsoluteMaxScanlines)
				}

				// change top scanline if it hasn't been changed recently
				if tv.state.vsync.topScanline == 0 {
					// take into account frame stability and whether the 'synced on
					// start' preference has been set
					if tv.state.frameInfo.Stable || !tv.env.Prefs.TV.VSYNCsyncedOnStart.Get().(bool) {
						tv.state.vsync.topScanline = (specification.AbsoluteMaxScanlines - tv.state.frameInfo.VisibleBottom) / 2
					}
				}
			}
		}

		// we must use TotalScanlines for the refresh rate calculation. the
		// tv.state.scanlines value (ie. the current scanline value) will not
		// work. it is true that the current scanline and TotalScanlines will be
		// same but not when the screen is unsynced, which in this codepath we
		// will be
		tv.state.frameInfo.RefreshRate = tv.state.frameInfo.Spec.HorizontalScanRate / float32(tv.state.frameInfo.TotalScanlines)
		tv.setRefreshRate(tv.state.frameInfo.RefreshRate)
		tv.state.frameInfo.Jitter = true
	} else {
		tv.state.frameInfo.Jitter = false
	}

	// prepare for next frame
	tv.state.frameNum++

	// handle scanline value. this is based on the current 'top scanline' value
	// if this is a frame as a result of a valid VSYNC
	if tv.state.fromVSYNC {
		// once the screen has synced then visible top comes into play and we
		// need to make sure there's no residual signal at the beginning of the
		// signal array
		for i := 0; i <= tv.state.vsync.topScanline*specification.ClksScanline; i++ {
			// see comment below about nullifying signals at the end of the frame
			tv.signals[i] = signal.SignalAttributes{
				Index: signal.NoSignal,
				Color: signal.ZeroBlack,
			}
		}

		tv.state.scanline = tv.state.vsync.topScanline

		// recover from screen roll by altering 'top scanline' value
		if tv.state.vsync.topScanline > 0 {
			if tv.state.frameInfo.FromVSYNC {
				tv.state.vsync.topScanline = int(float32(tv.state.vsync.topScanline) * recoverySpeed)
			} else {
				tv.state.vsync.topScanline = int(float32(tv.state.vsync.topScanline) * recoverySpeed * 0.5)
			}
		}
	} else {
		tv.state.scanline = 0
	}

	// note VSYNC information and update VSYNC history
	tv.state.frameInfo.TopScanline = tv.state.scanline
	tv.state.frameInfo.FromVSYNC = tv.state.fromVSYNC
	tv.state.frameInfo.VSYNCscanline = tv.state.vsync.startScanline
	tv.state.frameInfo.VSYNCclock = tv.state.vsync.startClock
	tv.state.frameInfo.VSYNCcount = tv.state.vsync.activeScanlineCount

	// reset fromVSYNC latch and failedVSYNC latch
	tv.state.fromVSYNC = false
	tv.state.failedVSYNC = false

	// nullify unused signals at end of frame
	for i := tv.currentSignalIdx; i < len(tv.signals); i++ {
		// ideally, we should just be able to set the Index field to NoSignal
		//
		// however, a signal renderer may choose to process a signal even when
		// the NoSignal index is present. for these cases, we need to nullify
		// the entire entry
		tv.signals[i] = signal.SignalAttributes{
			Index: signal.NoSignal,
			Color: signal.ZeroBlack,
		}
	}

	// push pending pixels and audio data
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
	tv.lmtr.CheckFrame()

	// measure frame rate
	tv.lmtr.MeasureActual()

	// signal index at beginning of new frame
	tv.firstSignalIdx = tv.state.clock + (tv.state.scanline * specification.ClksScanline)

	return nil
}

func (tv *Television) renderSignals() error {
	// do not render pixels if emulation is in the rewinding state
	if tv.emulationState != govern.Rewinding {
		for _, r := range tv.renderers {
			err := r.SetPixels(tv.signals, tv.currentSignalIdx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SetSpec sets the tv specification. Used when a new cartridge is attached to
// the console and when the user explicitely requests a change.
func (tv *Television) SetSpec(spec string) error {
	spec, ok := specification.NormaliseReqSpecID(spec)
	if !ok {
		return fmt.Errorf("television: unsupported spec (%s)", spec)
	}
	tv.setSpec(spec)
	return nil
}

func (tv *Television) setSpec(spec string) {
	tv.state.setSpec(spec)
	tv.setRefreshRate(tv.state.frameInfo.Spec.RefreshRate)
	if tv.realTimeMixer != nil {
		tv.realTimeMixer.SetSpec(tv.state.frameInfo.Spec)
	}
}

// setRefreshRate of TV. also calls the SetClockSpeed() function in the vcs
// interface
func (tv *Television) setRefreshRate(rate float32) {
	tv.lmtr.SetRefreshRate(rate)
	if tv.vcs != nil {
		tv.vcs.SetClockSpeed(tv.state.frameInfo.Spec)
	}
	tv.audioSignalLimit = baselineAudioSignalLimit
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
		tv.lmtr.MeasureActual()

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

// GetResetSpecID returns the specification that the television resets to. This
// is useful for learning more about how the television was created
func (tv *Television) GetResetSpecID() string {
	return tv.state.resetSpec
}

// IsAutoSpec returns true if the requested specification was "AUTO"
func (tv *Television) IsAutoSpec() bool {
	return tv.state.autoSpec
}

// GetFrameInfo returns the television's current frame information.
func (tv *Television) GetFrameInfo() frameinfo.Current {
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

func (tv *Television) IsFrameNum(frame int) bool {
	return tv.state.frameNum == frame
}
