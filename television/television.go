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
	"strings"

	"github.com/jetsetilly/gopher2600/errors"
)

// the number of times we must see new top/bottom scanline in the
// resize-window before we accept the new value
const resizeThreshold = 10

// the number of frames that (speculative) top and bottom values must be steady
// before we accept the frame characteristics
const stabilityThreshold = 15

// the number of scanlines required to be seen in the frame before we consider
// the tv to be operating "out of spec"
//
// this value is ridiculously wrong but I don't have any good reason for any
// other value. It's only ever really a issue when the cartridge is starting up
// but still, it would be nice to have a value with some sort of pedigree
const excessiveScanlines = 10000

// for the purposes of frame size detection, we should consider the first
// handful of frames to be unreliable
const unreliableFrames = 4

// television is a reference implementation of the Television interface. In all
// honesty, it's most likely the only implementation required.
type television struct {
	// television specification (NTSC or PAL)
	spec *Specification

	// spec on creation ID is the string that was to ID the television
	// type/spec on creation. because the actual spec can change, the ID field
	// of the Specification type can not be used for things like regression
	// test recreation etc.
	specIDOnCreation string

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
	lastSignal SignalAttributes

	// vsyncCount records the number of consecutive colorClocks the vsync signal
	// has been sustained. we use this to help correctly implement vsync.
	vsyncCount int

	// top and bottom of screen as detected by vblank/color signal
	top    int
	bottom int

	// resizer functionality
	resizer resizer

	// the number of frames the tv's top/bottom scanlines have remained the
	// same (ie. not changed). stability count is increased every frame if it
	// has not yet reached the stability threshold. the function IsStable()
	// reports true if stability threshold has been reached
	//
	// if stability has not been reached the counter is reset whenever the top
	// and bottom scanlines look like they might change
	stabilityCt int

	// has the tv frame ever been "out of spec"
	outOfSpec bool

	// the key color keeps track of whether the color signal changes over the
	// course of a scanline. if the color signal never changes (changing from
	// VideoBlack being the exception) then a resize event does not occur.
	//
	// A good example of this system in action is Tapper. if we didn't monitor
	// the "key color" the screen would be much larger than it needs to be.
	key    bool
	keyCol ColorSignal

	// framerate limiter
	lmtr limiter

	// whether to use the FPS value given in the TV specification
	lmtrSpec bool

	// list of renderer implementations to consult
	renderers []PixelRenderer

	// list of audio mixers to consult
	mixers []AudioMixer
}

// NewTelevision creates a new instance of the television type, satisfying the
// Television interface.
func NewTelevision(spec string) (Television, error) {
	tv := &television{
		specIDOnCreation: strings.ToUpper(spec),
	}

	// initialise resizer
	tv.resizer.reset(tv)

	// set specification
	err := tv.SetSpec(tv.specIDOnCreation)
	if err != nil {
		return nil, err
	}

	// initialise frame rate limiter
	tv.lmtr.init()
	tv.SetFPS(-1)

	// empty list of renderers
	tv.renderers = make([]PixelRenderer, 0)

	return tv, nil
}

func (tv television) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("FR=%04d SL=%03d HP=%03d", tv.frameNum, tv.scanline, tv.horizPos-HorizClksHBlank))
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

// Reset implements the Television interface.
func (tv *television) Reset() error {

	// we definitely do not call this on television initialisation because the
	// rest of the system may not be yet be in a suitable state

	err := tv.SetSpec(tv.specIDOnCreation)
	if err != nil {
		return err
	}

	tv.horizPos = 0
	tv.frameNum = 0
	tv.scanline = 0
	tv.vsyncCount = 0
	tv.lastSignal = SignalAttributes{}

	tv.top = tv.spec.ScanlineTop
	tv.bottom = tv.spec.ScanlineBottom

	tv.stabilityCt = 0
	tv.outOfSpec = false
	tv.key = false
	tv.keyCol = 0

	tv.resizer.reset(tv)
	tv.resizer.resize = true
	if err := tv.resizer.setSize(tv); err != nil {
		return err
	}

	return nil
}

// Signal implements the Television interface
func (tv *television) Signal(sig SignalAttributes) error {
	tv.horizPos++

	// once we reach the scanline's back-porch we'll reset the horizPos counter
	// and wait for the HSYNC signal. we do this so that the front-porch and
	// back-porch are 'together' at the beginning of the scanline. this isn't
	// strictly technically correct but it's convenient to think about
	// scanlines in this way (rather than having a split front and back porch)
	if tv.horizPos >= HorizClksScanline {
		tv.horizPos = 0
		tv.scanline++

		// checkRate evey scanline. see checkRate() commentary for why this is
		tv.lmtr.checkRate()

		if tv.scanline <= tv.spec.ScanlinesTotal {
			err := tv.newScanline(sig.VBlank)
			if err != nil {
				return err
			}
		} else {
			if tv.IsStable() {
				err := tv.newFrame()
				if err != nil {
					return err
				}
			} else if tv.scanline > excessiveScanlines {
				// it looks like the ROM isn't going to send a VSYNC signal any
				// time soon so we must fake the stabilityCt
				//
				// see test rom 'test-ane.bin' for an example of this
				tv.stabilityCt = stabilityThreshold
				err := tv.newFrame()
				if err != nil {
					return err
				}

				tv.outOfSpec = true
			}
		}

	}

	// check vsync signal at the time of the flyback
	//
	// !!TODO: replace VSYNC signal with extended HSYNC signal
	if sig.VSync && !tv.lastSignal.VSync {
		tv.vsyncCount = 0

	} else if !sig.VSync && tv.lastSignal.VSync {
		if tv.vsyncCount > 0 {
			err := tv.newFrame()
			if err != nil {
				return err
			}
		}
	}

	// we've "faked" the flyback signal above when horizPos reached
	// horizClksScanline. we need to handle the real flyback signal however, by
	// making sure we're at the correct horizPos value.  if horizPos doesn't
	// equal 16 at the front of the HSYNC or 36 at then back of the HSYNC, then
	// it indicates that the RSYNC register was used last scanline.
	if sig.HSync && !tv.lastSignal.HSync {
		tv.horizPos = 16

		// count vsync lines at start of hsync
		if sig.VSync || tv.lastSignal.VSync {
			tv.vsyncCount++
		}
	}
	if !sig.HSync && tv.lastSignal.HSync {
		tv.horizPos = 36
	}

	// doing nothing with CBURST signal

	// decode color using the regular color signal
	col := tv.spec.getColor(sig.Pixel)
	for f := range tv.renderers {
		err := tv.renderers[f].SetPixel(tv.horizPos, tv.scanline,
			col.R, col.G, col.B,
			sig.VBlank)
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

	// update resizing event information
	if tv.auto {
		tv.resizer.check(tv, sig)
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
	tv.lastSignal = sig

	return nil
}

func (tv *television) newScanline(vblank bool) error {
	// reset key color check
	if !vblank {
		tv.key = true
		tv.keyCol = VideoBlack
	}

	// notify renderers of new scanline
	for f := range tv.renderers {
		err := tv.renderers[f].NewScanline(tv.scanline)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tv *television) newFrame() error {
	// reset key color check
	tv.key = true
	tv.keyCol = VideoBlack

	// check to see if we should flip to PAL specficiation
	if tv.auto && tv.spec != SpecPAL &&
		tv.scanline >= maxNTSCscanlines &&
		tv.frameNum > unreliableFrames {

		// flip to PAL specifcation
		tv.SetSpec("PAL")
		tv.resizer.resize = true
	}

	// perform resize if necessary
	if err := tv.resizer.setSize(tv); err != nil {
		return err
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
		return tv.horizPos - HorizClksHBlank, nil
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
		tv.spec = SpecNTSC
		tv.auto = true

	default:
		return errors.New(errors.Television, fmt.Sprintf("unsupported tv specifcation (%s)", spec))
	}

	tv.top = tv.spec.ScanlineTop
	tv.bottom = tv.spec.ScanlineBottom

	return nil
}

// SpecIDOnCreation implements the Television interface
func (tv *television) SpecIDOnCreation() string {
	return tv.specIDOnCreation
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

// SetFPSCap implements the Television interface. Reasons for turning the cap
// off include performance measurement. The debugger also turns the cap off and
// replaces it with its own. The FPS limiter in this television implementation
// works at the frame level which is not fine grained enough for effective
// limiting of rates less than 1fps.
func (tv *television) SetFPSCap(limit bool) {
	tv.lmtr.limit = limit
}

// SetFPS implements the Television interface. A negative value resets the FPS
// to the specification's ideal value.
func (tv *television) SetFPS(fps float32) {
	if fps == -1 {
		fps = tv.spec.FramesPerSecond
	}
	tv.lmtr.setRate(fps, tv.spec.ScanlinesTotal)
}

// GetReqFPS implements the Television interface
func (tv *television) GetReqFPS() float32 {
	return tv.lmtr.requested
}

// GetActualFPS implements the Television interface. Note that FPS measurement
// still works even when frame capping is disabled.
func (tv *television) GetActualFPS() float32 {
	return tv.lmtr.actual
}

// GetLastSignal implements the Television interface
func (tv *television) GetLastSignal() SignalAttributes {
	return tv.lastSignal
}

// resizer abstractifies the information and tasks required to set the
// television screen to the right size
type resizer struct {
	// the top and bottom values can change but we don't want to resize the
	// screen by accident.
	//
	// the following fields help detect the occasions when the screen should be
	// resized. this information is the forwarded to the attached pixel
	// renderers.
	//
	// there are three sets of fields. one for the top scanline and one for the
	// bottom scanline. and also fields which keep track of the color signal.
	//
	// the top and bot fields record the most extreme value seen yet.
	// resizeTopCt and resizeBotCt record how many times that extreme value
	// hase been seen. lastly, resizeTopFr and resizeBotFr records which frame
	// the extremity was last seen - we don't want to count that we have seen
	// the new scanline every signal of the scanline.
	top      int
	topCt    int
	resizeFr int
	bot      int
	botCt    int
	botFr    int

	// resize event should take place at earliest convenient time
	resize bool
}

func (rz *resizer) reset(tv *television) {
	rz.top = -1
	rz.topCt = 0
	rz.resizeFr = 0
	rz.bot = -1
	rz.botCt = 0
	rz.botFr = 0
	rz.resize = false
	rz.setSize(tv)
}

// check to see if the VCS is trying to draw out of the current screen
// boundaries
func (rz *resizer) check(tv *television, sig SignalAttributes) {

	// take into account the VBlank signal and whether the color signal is inconsistent.
	//
	// we also want to ignore the first few frames of the session because may
	// give unreliable information with regards to the size of the frame
	//
	// we also don't ever want to resize "out of spec" tv frames
	if sig.VBlank || tv.key || tv.frameNum <= unreliableFrames || tv.outOfSpec {
		return
	}

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
	if (rz.top != -1 && tv.scanline < rz.top) || (rz.top == -1 && tv.scanline < tv.top) {
		rz.topCt = 0
		rz.top = tv.scanline
		rz.resizeFr = tv.frameNum

		// if stability has not yet been reached, reset stability count
		if !tv.IsStable() {
			tv.stabilityCt = 0
		}
	} else if tv.frameNum > rz.resizeFr && tv.scanline == rz.top {
		rz.resizeFr = tv.frameNum
		rz.topCt++
		if rz.topCt >= resizeThreshold {
			tv.top = rz.top
			rz.resize = true
			rz.topCt = 0
			rz.top = -1
		}
	}

	if (rz.bot != -1 && tv.scanline > rz.bot) || (rz.bot == -1 && tv.scanline > tv.bottom) {
		rz.botFr = tv.frameNum
		rz.botCt = 0
		rz.bot = tv.scanline

		// if stability has not yet been reached, reset stability count
		if !tv.IsStable() {
			tv.stabilityCt = 0
		}
	} else if tv.frameNum > rz.botFr && tv.scanline == rz.bot {
		rz.botFr = tv.frameNum
		rz.botCt++
		if rz.botCt >= resizeThreshold {
			tv.bottom = rz.bot
			rz.resize = true
			rz.botCt = 0
			rz.bot = -1
		}
	}
}

func (rz *resizer) setSize(tv *television) error {
	if rz.resize {
		for f := range tv.renderers {
			err := tv.renderers[f].Resize(tv.top, tv.bottom-tv.top+1)
			if err != nil {
				return err
			}
		}

		// change fps
		if tv.lmtrSpec {
			tv.SetFPS(tv.spec.FramesPerSecond)
		}

		rz.resize = false
	}

	return nil
}
