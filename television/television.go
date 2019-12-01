package television

import (
	"fmt"
	"gopher2600/errors"
	"strings"
)

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

	// the following values are used for stability detection. we could possibly
	// define a separate type for all of these.

	// top and bottom of screen as detected by vblank/color signal
	top    int
	bottom int

	// new top and bottom values if stability threshold is met
	speculativeTop    int
	speculativeBottom int

	// top and bottom as reckoned by the current frame - reset at the moment
	// when a new frame is detected
	thisTop    int
	thisBottom int

	// a frame has to be stable (speculative top and bottom unchanged) for a
	// number of frames (stable threshold) before we accept that it is a true
	// representation of frame dimensions
	stability int
}

// the number of frames that (speculative) top and bottom values must be steady
// before we accept the frame characteristics
const stabilityThreshold = 5

// NewTelevision creates a new instance of StellaTelevision for a
// minimalist implementation of a televsion for the VCS emulation
func NewTelevision(tvType string) (Television, error) {
	tv := &television{}

	switch strings.ToUpper(tvType) {
	case "NTSC":
		tv.spec = SpecNTSC
	case "PAL":
		tv.spec = SpecPAL
	case "AUTO":
		tv.spec = SpecNTSC
		tv.auto = true
	default:
		return nil, errors.New(errors.Television, fmt.Sprintf("unsupported tv type (%s)", tvType))
	}

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
	tv.prevSignal = SignalAttributes{Pixel: VideoBlack}

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
			// repeat last scanline over and over
			tv.scanline = tv.spec.ScanlinesTotal
		}

	} else {
		tv.horizPos++
		if tv.horizPos > HorizClksScanline {
			return errors.New(errors.Television, "no flyback signal")
		}
	}

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
		if tv.vsyncCount >= tv.spec.scanlinesVSync {
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
	red, green, blue := getAltColor(sig.AltPixel)
	for f := range tv.renderers {
		err := tv.renderers[f].SetAltPixel(x, y, red, green, blue, sig.VBlank)
		if err != nil {
			return err
		}
	}

	// decode color using the regular color signal
	red, green, blue = getColor(tv.spec, sig.Pixel)
	for f := range tv.renderers {
		err := tv.renderers[f].SetPixel(x, y, red, green, blue, sig.VBlank)
		if err != nil {
			return err
		}
	}

	// push screen boundaries outward using vblank and color signal to help us
	if !sig.VBlank && red != 0 && green != 0 && blue != 0 {
		if tv.scanline < tv.thisTop {
			tv.thisTop = tv.scanline
		}
		if tv.scanline > tv.thisBottom {
			tv.thisBottom = tv.scanline
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

func (tv *television) stabilise() (bool, error) {
	if tv.frameNum <= 1 || (tv.thisTop == tv.top && tv.thisBottom == tv.bottom) {
		return false, nil
	}

	// if top and bottom has changed this frame update speculative values
	if tv.thisTop != tv.speculativeTop || tv.thisBottom != tv.speculativeBottom {
		tv.speculativeTop = tv.thisTop
		tv.speculativeBottom = tv.thisBottom
		return false, nil
	}

	// increase stability value until we reach threshold
	if !tv.IsStable() {
		tv.stability++
		return false, nil
	}

	// accept speculative values
	tv.top = tv.speculativeTop
	tv.bottom = tv.speculativeBottom

	if tv.spec == SpecNTSC && tv.bottom-tv.top >= SpecPAL.ScanlinesVisible {
		tv.spec = SpecPAL

		// reset top/bottom to ideals of new spec. they may of course be
		// pushed outward in subsequent frames
		tv.top = tv.spec.ScanlineTop
		tv.bottom = tv.spec.ScanlineBottom
	}

	for f := range tv.renderers {
		err := tv.renderers[f].Resize(tv.top, tv.bottom-tv.top+1)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (tv *television) newFrame() error {
	_, err := tv.stabilise()
	if err != nil {
		return err
	}

	// new frame
	tv.frameNum++
	tv.scanline = 0
	tv.thisTop = tv.top
	tv.thisBottom = tv.bottom

	// call new frame for all renderers
	for f := range tv.renderers {
		err = tv.renderers[f].NewFrame(tv.frameNum)
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

// GetSpec implements the Television interface
func (tv television) GetSpec() *Specification {
	return tv.spec
}

// IsStable implements the Television interface
func (tv television) IsStable() bool {
	return tv.stability >= stabilityThreshold
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
