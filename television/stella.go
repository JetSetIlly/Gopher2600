package television

import (
	"fmt"
	"gopher2600/errors"
	"strings"
)

// StellaTelevision is the minimalist implementation of the Television
// interface. It is so called because the reporting of the TV state, via
// GetState(), is meant to mirror exactly the state as reported by the stella
// emulator. The intention is to make it easier to perform A/B testing.
//
// To make the state reporting as intuitive as possible, StellaTelevision makes
// use of the HSyncSimple sigal attribute (see SignalAttributes type in the
// television package for details). Consequently, calls to NewScanline() for
// any attached renderers, are made when the HSyncSimple signal is recieved.
// This will have an effect on how the renderer displays off screen information
// (if it chooses to that is).
type StellaTelevision struct {
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

// NewStellaTelevision creates a new instance of StellaTelevision for a
// minimalist implementation of a televsion for the VCS emulation
func NewStellaTelevision(tvType string) (*StellaTelevision, error) {
	btv := new(StellaTelevision)

	switch strings.ToUpper(tvType) {
	case "NTSC":
		btv.spec = SpecNTSC
	case "PAL":
		btv.spec = SpecPAL
	case "AUTO":
		btv.spec = SpecNTSC
		btv.auto = true
	default:
		return nil, errors.New(errors.StellaTelevision, fmt.Sprintf("unsupported tv type (%s)", tvType))
	}

	// empty list of renderers
	btv.renderers = make([]PixelRenderer, 0)

	// initialise TVState
	err := btv.Reset()
	if err != nil {
		return nil, err
	}

	return btv, nil
}

func (btv StellaTelevision) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("FR=%d SL=%d", btv.frameNum, btv.scanline))
	s.WriteString(fmt.Sprintf(" HP=%d", btv.horizPos))
	return s.String()
}

// AddPixelRenderer implements the Television interface
func (btv *StellaTelevision) AddPixelRenderer(r PixelRenderer) {
	btv.renderers = append(btv.renderers, r)
}

// AddAudioMixer implements the Television interface
func (btv *StellaTelevision) AddAudioMixer(m AudioMixer) {
	btv.mixers = append(btv.mixers, m)
}

// Reset implements the Television interface
func (btv *StellaTelevision) Reset() error {
	btv.horizPos = -ClocksPerHblank
	btv.frameNum = 0
	btv.scanline = 0
	btv.vsyncCount = 0
	btv.prevSignal = SignalAttributes{Pixel: VideoBlack}

	btv.top = btv.spec.ScanlineTop
	btv.bottom = btv.spec.ScanlineBottom

	return nil
}

// Signal implements the Television interface
func (btv *StellaTelevision) Signal(sig SignalAttributes) error {
	// the following condition detects a new scanline by looking for the
	// non-textbook HSyncSimple signal
	//
	// see SignalAttributes type definition for notes about the HSyncSimple
	// attribute
	if sig.HSyncSimple && !btv.prevSignal.HSyncSimple {
		btv.horizPos = -ClocksPerHblank
		btv.scanline++

		if btv.scanline <= btv.spec.ScanlinesTotal {
			// when observing Stella we can see that on the first frame (frame
			// number zero) a new frame is triggered when the scanline reaches
			// 51.  it does this with every ROM and regardless of what signals
			// have been sent.
			//
			// I'm not sure why it does this but we emulate the behaviour here
			// in order to facilitate A/B testing.
			if btv.frameNum == 0 && btv.scanline > 50 {
				btv.scanline = 0
				btv.frameNum++

				// notify renderers of new frame
				for f := range btv.renderers {
					err := btv.renderers[f].NewFrame(btv.frameNum)
					if err != nil {
						return err
					}
				}
			} else {
				// notify renderers of new scanline
				for f := range btv.renderers {
					err := btv.renderers[f].NewScanline(btv.scanline)
					if err != nil {
						return err
					}
				}
			}
		} else {
			// repeat last scanline over and over
			btv.scanline = btv.spec.ScanlinesTotal
		}

	} else {
		btv.horizPos++
		if btv.horizPos > ClocksPerScanline {
			return errors.New(errors.StellaTelevision, "no flyback signal")
		}
	}

	// simple vsync implementation. when compared to the HSync detection above,
	// the following is correct (front porch at the end of the display and back
	// porch at the beginning). it is also in keeping with how Stella counts
	// scanlines, meaning A/B testing is relatively straightforward.
	if sig.VSync {
		// if this a new vsync sequence note the horizontal position
		if !btv.prevSignal.VSync {
			btv.vsyncPos = btv.horizPos
		}
		// bump the vsync count whenever vsync is set
		btv.vsyncCount++
	} else if btv.prevSignal.VSync {
		// if vsync has just be turned off then check that it has been held for
		// the requisite number of scanlines for a new frame to be started
		if btv.vsyncCount >= btv.spec.ScanlinesPerVSync {
			err := btv.newFrame()
			if err != nil {
				return err
			}
		}

		// reset vsync counter when vsync signal is dropped
		btv.vsyncCount = 0
	}

	// record the current signal settings so they can be used for reference
	btv.prevSignal = sig

	// current coordinates
	x := btv.horizPos + ClocksPerHblank
	y := btv.scanline

	// decode color using the regular color signal
	red, green, blue := getColor(btv.spec, sig.Pixel)
	for f := range btv.renderers {
		err := btv.renderers[f].SetPixel(x, y, red, green, blue, sig.VBlank)
		if err != nil {
			return err
		}
	}

	// push screen boundaries outward using vblank and color signal to help us
	if !sig.VBlank && red != 0 && green != 0 && blue != 0 {
		if btv.scanline < btv.thisTop {
			btv.thisTop = btv.scanline
		}
		if btv.scanline > btv.thisBottom {
			btv.thisBottom = btv.scanline
		}
	}

	// decode color using the alternative color signal
	red, green, blue = getAltColor(sig.AltPixel)
	for f := range btv.renderers {
		err := btv.renderers[f].SetAltPixel(x, y, red, green, blue, sig.VBlank)
		if err != nil {
			return err
		}
	}

	// mix audio on UpdateAudio signal
	if sig.UpdateAudio {
		for f := range btv.mixers {
			err := btv.mixers[f].SetAudio(sig.Audio)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (btv *StellaTelevision) stabilise() (bool, error) {
	if btv.frameNum <= 1 || (btv.thisTop == btv.top && btv.thisBottom == btv.bottom) {
		return false, nil
	}

	// if top and bottom has changed this frame update speculative values
	if btv.thisTop != btv.speculativeTop || btv.thisBottom != btv.speculativeBottom {
		btv.speculativeTop = btv.thisTop
		btv.speculativeBottom = btv.thisBottom
		return false, nil
	}

	// increase stability value until we reach threshold
	if !btv.IsStable() {
		btv.stability++
		return false, nil
	}

	// accept speculative values
	btv.top = btv.speculativeTop
	btv.bottom = btv.speculativeBottom

	if btv.spec == SpecNTSC && btv.bottom-btv.top >= SpecPAL.ScanlinesPerVisible {
		btv.spec = SpecPAL

		// reset top/bottom to ideals of new spec. they may of course be
		// pushed outward in subsequent frames
		btv.top = btv.spec.ScanlineTop
		btv.bottom = btv.spec.ScanlineBottom
	}

	for f := range btv.renderers {
		err := btv.renderers[f].Resize(btv.top, btv.bottom-btv.top+1)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (btv *StellaTelevision) newFrame() error {
	_, err := btv.stabilise()
	if err != nil {
		return err
	}

	// new frame
	btv.frameNum++
	btv.scanline = 0
	btv.thisTop = btv.top
	btv.thisBottom = btv.bottom

	// call new frame for all renderers
	for f := range btv.renderers {
		err = btv.renderers[f].NewFrame(btv.frameNum)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetState implements the Television interface
func (btv *StellaTelevision) GetState(request StateReq) (int, error) {
	switch request {
	default:
		return 0, errors.New(errors.UnknownTVRequest, request)
	case ReqFramenum:
		return btv.frameNum, nil
	case ReqScanline:
		return btv.scanline, nil
	case ReqHorizPos:
		return btv.horizPos, nil
	}
}

// GetSpec implements the Television interface
func (btv StellaTelevision) GetSpec() *Specification {
	return btv.spec
}

// IsStable implements the Television interface
func (btv StellaTelevision) IsStable() bool {
	return btv.stability >= stabilityThreshold
}
