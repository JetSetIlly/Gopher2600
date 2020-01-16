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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package television

import (
	"fmt"
	"gopher2600/errors"
	"strings"
)

// the number of times we must see new top/bottom scanline in the
// resize-window before we accept the new value
const resizeThreshold = 10

// the number of frames that (speculative) top and bottom values must be steady
// before we accept the frame characteristics
const stabilityThreshold = 15

// the number of scanlines past the NTSC limit before the specification flips
// to PAL (auto flag permitting)
const overageNTSC = 10

// for the purposes of frame size detection, we should consider the first
// handful of frames to be unreliable
const unreliableFrames = 4

// television is a reference implementation of the Television interface. In all
// honesty, it's most likely the only implementation required.
type television struct {
	// television specification (NTSC or PAL)
	spec *Specification

	// auto flag indicates that the tv type/specification should switch if it
	// appears to be outside of the current spec.
	//
	// in practice this means that if auto is true then we start with the NTSC
	// spec and move to PAL if the number of scanlines exceeds the NTSC maximum
	auto bool

	// state of the television
	//	- the current horizontal position. the position where the next pixel will be
	//  drawn. also used to check we're receiving the correct signals at the
	//  correct time.
	horizPos int
	//	- the current frame
	frameNum int
	//	- the current scanline number
	scanline int

	// record of signal attributes from the last call to Signal()
	prevSignal SignalAttributes

	// vsyncCount records the number of consecutive colorClocks the vsync signal
	// has been sustained. we use this to help correctly implement vsync.
	vsyncCount int
	vsyncPos   int

	// list of renderer implementations to consult
	renderers []PixelRenderer

	// list of audio mixers to consult
	mixers []AudioMixer

	// top and bottom of screen as detected by vblank/color signal
	top    int
	bottom int

	// the number of frames the tv's top/bottom scanlines have remained the
	// same (ie. not changed). stability count is increased every frame if it
	// has not yet reached the stability threshold. the function IsStable()
	// reports true if stability threshold has been reached
	//
	// if stability has not been reached the counter is reset whenever the top
	// and bottom scanlines look like they might change
	stabilityCt int

	// the top and bottom values can change but we don't want to resize the
	// screen by accident.
	//
	// the following fields help detect the real occasions when the screen
	// should be resized. this information is the forwarded to the attached
	// pixel renderers.
	//
	// there are three sets of fields. one for the top scanline and one for the
	// bottom scanline. and also fields which keep track of the color signal.
	//
	// the resizeTop and resizeBot record the most extreme value seen yet.
	// resizeTopCt and resizeBotCt record how many times that extreme value
	// hase been seen. lastly, resizeTopFr and resizeBotFr records which frame
	// the extremity was last seen - we don't want to count that we have seen
	// the new scanline every signal of the scanline.
	resizeTop   int
	resizeTopCt int
	resizeTopFr int
	resizeBot   int
	resizeBotCt int
	resizeBotFr int
	resize      bool

	// the key color keeps track of whether the color signal changes over the
	// course of a scanline. if the color signal never changes (changing from
	// VideoBlack being the exception) then a resize event does not occur.
	//
	// A good example of this system in action is Tapper. if we didn't monitor
	// the "key color" the screen would be much larger than it needs to be.
	key    bool
	keyCol ColorSignal
}

// NewTelevision creates a new instance of the television type, satisfying the
// Television interface.
func NewTelevision(spec string) (Television, error) {
	tv := &television{
		resizeTop: -1,
		resizeBot: -1,
	}

	tv.SetSpec(spec)

	// empty list of renderers
	tv.renderers = make([]PixelRenderer, 0)

	// initialise TVState
	err := tv.Reset()
	if err != nil {
		return nil, err
	}

	return tv, nil
}

func (tv television) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("FR=%d SL=%d", tv.frameNum, tv.scanline))
	s.WriteString(fmt.Sprintf(" HP=%d", tv.horizPos))
	return s.String()
}

// AddPixelRenderer implements the Television interface
func (tv *television) AddPixelRenderer(r PixelRenderer) {
	tv.renderers = append(tv.renderers, r)
}

// AddAudioMixer implements the Television interface
func (tv *television) AddAudioMixer(m AudioMixer) {
	tv.mixers = append(tv.mixers, m)
}

// Reset implements the Television interface
func (tv *television) Reset() error {
	tv.horizPos = -HorizClksHBlank
	tv.frameNum = 0
	tv.scanline = 0
	tv.vsyncCount = 0
	tv.prevSignal = SignalAttributes{}

	tv.top = tv.spec.ScanlineTop
	tv.bottom = tv.spec.ScanlineBottom

	return nil
}

// Signal implements the Television interface
func (tv *television) Signal(sig SignalAttributes) error {
	// the following condition detects a new scanline by looking for the
	// non-textbook HSyncSimple signal
	//
	// see SignalAttributes type definition for notes about the HSyncSimple
	// attribute
	if sig.HSyncSimple && !tv.prevSignal.HSyncSimple {
		tv.horizPos = -HorizClksHBlank
		tv.scanline++

		// reset key color check for the new scanline
		tv.key = true
		tv.keyCol = VideoBlack

		if tv.scanline <= tv.spec.ScanlinesTotal {
			// when observing Stella we can see that on the first frame (frame
			// number zero) a new frame is triggered when the scanline reaches
			// 51.  it does this with every ROM and regardless of what signals
			// have been sent.
			//
			// I'm not sure why it does this but we emulate the behaviour here
			// in order to facilitate A/B testing.
			if tv.frameNum == 0 && tv.scanline > 50 {
				tv.scanline = 0
				tv.frameNum++

				// notify renderers of new frame
				for f := range tv.renderers {
					err := tv.renderers[f].NewFrame(tv.frameNum)
					if err != nil {
						return err
					}
				}
			} else {
				// notify renderers of new scanline
				for f := range tv.renderers {
					err := tv.renderers[f].NewScanline(tv.scanline)
					if err != nil {
						return err
					}
				}
			}
		} else {
			// PAL detection condition:
			//   1. frame must be "unstable"
			//   2. not be the first frame (because ROMs can still be in the
			//       setup phae at this point)
			//   3. not be in PAL mode already
			//   4. have the auto flag set
			//   5. be more than 10 scanlines beyond the NTSC specification
			//
			// Specification detection only works from NTSC to PAL. A PAL frame
			// can never cause a flip to NTSC
			if !tv.IsStable() && tv.frameNum > 1 &&
				tv.spec != SpecPAL && tv.auto &&
				tv.scanline >= SpecNTSC.ScanlinesTotal+overageNTSC {

				tv.SetSpec("PAL")
				tv.resize = true
			}
		}

	} else {
		tv.horizPos++
		if tv.horizPos > HorizClksScanline {
			return errors.New(errors.Television, "no flyback signal")
		}
	}

	// not doing anything with the "real" hsync or colour burst signals

	// simple vsync implementation. when compared to the HSync detection above,
	// the following is correct (front porch at the end of the display and back
	// porch at the beginning). it is also in keeping with how Stella counts
	// scanlines, meaning A/B testing is relatively straightforward.
	if sig.VSync {
		// if this a new vsync sequence note the horizontal position
		if !tv.prevSignal.VSync {
			tv.vsyncPos = tv.horizPos
		}
		// bump the vsync count whenever vsync is set
		tv.vsyncCount++
	} else if tv.prevSignal.VSync {
		// if vsync has just be turned off then check that it has been held for
		// the requisite number of scanlines for a new frame to be started
		if tv.vsyncCount >= tv.spec.ScanlinesVSync {
			err := tv.newFrame()
			if err != nil {
				return err
			}
		}

		// reset vsync counter when vsync signal is dropped
		tv.vsyncCount = 0
	}

	// current coordinates
	x := tv.horizPos + HorizClksHBlank
	y := tv.scanline

	// decode color using the alternative color signal
	col := getAltColor(sig.AltPixel)
	for f := range tv.renderers {
		err := tv.renderers[f].SetAltPixel(x, y, col.red, col.green, col.blue, sig.VBlank)
		if err != nil {
			return err
		}
	}

	// decode color using the regular color signal
	col = getColor(tv.spec, sig.Pixel)
	for f := range tv.renderers {
		err := tv.renderers[f].SetPixel(x, y, col.red, col.green, col.blue, sig.VBlank)
		if err != nil {
			return err
		}
	}

	// check for color signal consistency
	if tv.key && sig.Pixel != VideoBlack {
		if tv.keyCol == VideoBlack {
			tv.keyCol = sig.Pixel
		} else if tv.keyCol != sig.Pixel {
			tv.key = false
		}
	}

	// check to see if the VCS is trying to draw out of the current screen
	// boundaries, taking into account the VBlank signal and whether the color
	// signal is inconsistent.
	//
	// we also want to ignore the first few frames of the session because may
	// give unreliable information with regards to the size of the frame
	if !sig.VBlank && !tv.key && tv.frameNum > unreliableFrames {
		// size detection:
		//
		// 1. if scanline is below/above current top/bottom or below/above current
		//          candidate values for top/bottom
		// 2. start a new count and consider current scanline as possibly the
		//          new top/bottom
		// 3. once the candidate value for the new top/bottom has been seen a
		//          certain number of times on different frames, then accept
		//          this as the new limit and set resize flag to true
		//
		// this is a little more complex that just looking for a stable value
		// that endures for a threshold number of frame. this is because some
		// ROMs never stabilise on a fixed value but otherwise consistently
		// draw outside of the currently defined area. for example, Frogger's
		// top scanline flutters between 35 and 40. we want it to settle on
		// scanline 35.
		if (tv.resizeTop != -1 && tv.scanline < tv.resizeTop) || (tv.resizeTop == -1 && tv.scanline < tv.top) {
			tv.resizeTopCt = 0
			tv.resizeTop = tv.scanline
			tv.resizeTopFr = tv.frameNum

			// if stability has not yet been reached, reset stability count
			if !tv.IsStable() {
				tv.stabilityCt = 0
			}
		} else if tv.frameNum > tv.resizeTopFr && tv.scanline == tv.resizeTop {
			tv.resizeTopFr = tv.frameNum
			tv.resizeTopCt++
			if tv.resizeTopCt >= resizeThreshold {
				tv.top = tv.resizeTop
				tv.resize = true
				tv.resizeTopCt = 0
				tv.resizeTop = -1
			}
		}

		if (tv.resizeBot != -1 && tv.scanline > tv.resizeBot) || (tv.resizeBot == -1 && tv.scanline > tv.bottom) {
			tv.resizeBotFr = tv.frameNum
			tv.resizeBotCt = 0
			tv.resizeBot = tv.scanline

			// if stability has not yet been reached, reset stability count
			if !tv.IsStable() {
				tv.stabilityCt = 0
			}
		} else if tv.frameNum > tv.resizeBotFr && tv.scanline == tv.resizeBot {
			tv.resizeBotFr = tv.frameNum
			tv.resizeBotCt++
			if tv.resizeBotCt >= resizeThreshold {
				tv.bottom = tv.resizeBot
				tv.resize = true
				tv.resizeBotCt = 0
				tv.resizeBot = -1
			}
		}
	}

	// mix audio
	if sig.AudioUpdate {
		for f := range tv.mixers {
			err := tv.mixers[f].SetAudio(sig.AudioData)
			if err != nil {
				return err
			}
		}
	}

	// record the current signal settings so they can be used for reference
	tv.prevSignal = sig

	return nil
}

func (tv *television) newFrame() error {
	// screen resizing has been requested
	if tv.resize {
		for f := range tv.renderers {
			err := tv.renderers[f].Resize(tv.top, tv.bottom-tv.top+1)
			if err != nil {
				return err
			}
		}
		tv.resize = false
	}

	// new frame
	tv.frameNum++
	tv.scanline = 0

	// if frame is not currently stable them increase stability count
	if !tv.IsStable() {
		tv.stabilityCt++
	}

	// call new frame for all renderers
	for f := range tv.renderers {
		err := tv.renderers[f].NewFrame(tv.frameNum)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetState implements the Television interface
func (tv *television) GetState(request StateReq) (int, error) {
	switch request {
	default:
		return 0, errors.New(errors.UnknownTVRequest, request)
	case ReqFramenum:
		return tv.frameNum, nil
	case ReqScanline:
		return tv.scanline, nil
	case ReqHorizPos:
		return tv.horizPos, nil
	}
}

// SetSpec implements the Television interface
func (tv *television) SetSpec(spec string) error {
	switch strings.ToUpper(spec) {
	case "NTSC":
		tv.spec = SpecNTSC
		tv.auto = false
	case "PAL":
		tv.spec = SpecPAL
		tv.auto = false
	case "AUTO":
		tv.auto = true

		// a tv.spec of nil means this is the first call of SetSpec() so
		// as well as setting the auto flag we need to specify a
		// specification
		if tv.spec == nil {
			tv.spec = SpecNTSC
		}

	default:
		return errors.New(errors.Television, fmt.Sprintf("unsupported tv specifcation (%s)", spec))
	}

	tv.top = tv.spec.ScanlineTop
	tv.bottom = tv.spec.ScanlineBottom

	return nil
}

// GetSpec implements the Television interface
func (tv television) GetSpec() *Specification {
	return tv.spec
}

// IsStable implements the Television interface
func (tv television) IsStable() bool {
	return tv.stabilityCt >= stabilityThreshold
}

// End implements the Television interface
func (tv television) End() error {
	var err error

	// call new frame for all renderers
	for f := range tv.renderers {
		err = tv.renderers[f].EndRendering()
	}

	// flush audio for all mixers
	for f := range tv.mixers {
		err = tv.mixers[f].EndMixing()
	}

	return err
}
