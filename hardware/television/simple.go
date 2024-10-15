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
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
)

type simple struct {
	vsyncActive       bool
	vsyncStartOnClock int
	vsyncClocks       int
	vsyncScanlines    int
}

// NewSimpleTelevision creates a Television instance but with the simple signal
// path enabled. Simple Televisions should only be used for raw testing of raw
// output from the emulated VCS
func NewSimpleTelevision(spec string) (*Television, error) {
	tv, err := NewTelevision(spec)
	if err != nil {
		return nil, err
	}
	tv.simple = &simple{}
	tv.signal = tv.signalSimple
	return tv, nil
}

// SetSimple switches the television instance between simple and normal operations
func (tv *Television) SetSimple(set bool) {
	if set {
		if tv.simple == nil {
			tv.simple = &simple{}
			tv.signal = tv.signalSimple
		}
	} else {
		if tv.simple != nil {
			tv.simple = nil
			tv.signal = tv.signalFull
		}
	}
}

// IsSimple returns true if the television is operating simply
func (tv *Television) IsSimple() bool {
	return tv.simple != nil
}

func (tv *Television) signalSimple(sig signal.SignalAttributes) {
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

		if tv.state.scanline >= specification.AbsoluteMaxScanlines {
			err := tv.newFrameSimple(false)
			if err != nil {
				logger.Log(tv.env, "Simple TV", err)
			}
		} else {
			// if we're not at end of screen then indicate new scanline
			err := tv.newScanline()
			if err != nil {
				logger.Log(tv.env, "Simple TV", err)
			}
		}
	}

	// count VSYNC clocks and scanlines
	if tv.simple.vsyncActive {
		if tv.state.clock == tv.simple.vsyncStartOnClock {
			tv.simple.vsyncScanlines++
		}
		tv.simple.vsyncClocks++
	}

	// check for change of VSYNC signal
	if sig.VSync != tv.state.lastSignal.VSync {
		if sig.VSync {
			// VSYNC has started
			tv.simple.vsyncActive = true
			tv.simple.vsyncScanlines = 0
			tv.simple.vsyncStartOnClock = tv.state.clock
		} else {
			// VSYNC has ended but we don't want to trigger a new frame unless
			// the VSYNC signal has been present for a minimum number of
			// clocks
			//
			// the value used hes is an absolute minimum value that is very
			// forgiving of a poorly constructed VSYNC signal. however, for the
			// purposes of the 'simple television' implementation this is
			// appropriate
			if tv.simple.vsyncClocks > 10 {
				err := tv.newFrameSimple(true)
				if err != nil {
					logger.Log(tv.env, "TV", err)
				}
			}
			tv.simple.vsyncActive = false
		}
		tv.simple.vsyncClocks = 0
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

	// render queued signals
	if tv.currentSignalIdx >= len(tv.signals) {
		err := tv.renderSignals()
		if err != nil {
			logger.Log(tv.env, "Simple TV", err)
		}
	}
}

func (tv *Television) newFrameSimple(fromVsync bool) error {
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
			if tv.state.reqSpecID == "AUTO" && tv.state.scanline > specification.PALTrigger {
				tv.setSpec("PAL")
			}
		case specification.SpecPAL.ID:
			if tv.state.reqSpecID == "AUTO" && tv.state.scanline <= specification.PALTrigger {
				tv.setSpec("NTSC")
			}
		}
	}

	// update frame number
	tv.state.frameInfo.FrameNum = tv.state.frameNum

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
		tv.state.frameInfo.RefreshRate = tv.state.frameInfo.Spec.HorizontalScanRate / float32(tv.state.scanline)
		tv.setRefreshRate(tv.state.frameInfo.RefreshRate)
		tv.state.frameInfo.Jitter = true
	} else {
		tv.state.frameInfo.Jitter = false
	}

	// prepare for next frame
	tv.state.frameNum++
	tv.state.scanline = 0

	// nullify unused signals at end of frame
	for i := tv.currentSignalIdx; i < len(tv.signals); i++ {
		tv.signals[i].Index = signal.NoSignal
	}

	// set pending pixels
	err := tv.renderSignals()
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
