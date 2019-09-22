package television

import (
	"fmt"
	"gopher2600/errors"
	"strings"
)

// StellaTelevision is the minimalist implementation of the Television interface - a
// television without a screen. Fuller implementations of the television can
// use this as the basis of the emulation by struct embedding
//
// the reporting of TV state is meant to mirror exactly, the state as reported
// by stella. this makes it easier to perform A/B testing
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
	//  - the number of scanlines past the specification limit. used to
	//  trigger a change of tv specification
	extraScanlines int

	// record of signal attributes from the last call to Signal()
	prevSignal SignalAttributes

	// vsyncCount records the number of consecutive colorClocks the vsync signal
	// has been sustained. we use this to help correctly implement vsync.
	vsyncCount int
	vsyncPos   int

	// the scanline at which the visible part of the screen begins and ends
	// - we start off with ideal values and push the screen outwards as
	// required
	visibleTop    int
	visibleBottom int

	// thisVisibleTop/Bottom records visible part of the screen (as described
	// above) during the current frame. we use these to update the real
	// variables at the end of a frame
	thisVisibleTop    int
	thisVisibleBottom int

	// list of renderer implementations to consult
	renderers []Renderer

	// list of audio mixers to consult
	mixers []AudioMixer
}

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
	btv.renderers = make([]Renderer, 0)

	// initialise TVState
	err := btv.Reset()
	if err != nil {
		return nil, err
	}

	return btv, nil
}

// MachineInfoTerse returns the television information in terse format
func (btv StellaTelevision) MachineInfoTerse() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("FR=%d SL=%d", btv.frameNum, btv.scanline))
	if btv.extraScanlines > 0 {
		s.WriteString(fmt.Sprintf(" [%d]", btv.extraScanlines))
	}
	s.WriteString(fmt.Sprintf(" HP=%d", btv.horizPos))
	return s.String()
}

// MachineInfo returns the television information in verbose format
func (btv StellaTelevision) MachineInfo() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("TV (%s)", btv.spec.ID))
	s.WriteString(fmt.Sprintf("\n   Frame: %d\n", btv.frameNum))
	s.WriteString(fmt.Sprintf("   Scanline: %d", btv.scanline))
	if btv.extraScanlines > 0 {
		s.WriteString(fmt.Sprintf(" [%d]\n", btv.extraScanlines))
	} else {
		s.WriteString("\n")
	}
	s.WriteString(fmt.Sprintf("   Horiz Pos: %d", btv.horizPos))
	return s.String()
}

// map String to MachineInfo
func (btv StellaTelevision) String() string {
	return btv.MachineInfo()
}

// AddRenderer adds a renderer implementation to the list
func (btv *StellaTelevision) AddRenderer(r Renderer) {
	btv.renderers = append(btv.renderers, r)
}

// AddMixer adds a renderer implementation to the list
func (btv *StellaTelevision) AddMixer(m AudioMixer) {
	btv.mixers = append(btv.mixers, m)
}

// Reset all the values for the television
func (btv *StellaTelevision) Reset() error {
	btv.horizPos = -ClocksPerHblank
	btv.frameNum = 0
	btv.scanline = 0
	btv.vsyncCount = 0
	btv.prevSignal = SignalAttributes{Pixel: VideoBlack}

	// default top/bottom to the "ideal" values
	btv.thisVisibleTop = btv.spec.ScanlinesTotal
	btv.thisVisibleBottom = 0

	return nil
}

func (btv *StellaTelevision) autoSpec() (bool, error) {
	if !btv.auto {
		return false, nil
	}

	if btv.spec == SpecPAL {
		return false, nil
	}

	btv.spec = SpecPAL
	for f := range btv.renderers {
		err := btv.renderers[f].ChangeTVSpec()
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// Signal is principle method of communication between the VCS and televsion
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
			// if we're above the scanline limit for the specification then don't
			// notify the renderers of a new scanline, instead repeat drawing to
			// the last scanline and note the number of "extra" scanlines we've
			// encountered
			btv.scanline = btv.spec.ScanlinesTotal
			btv.extraScanlines++
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
			btv.frameNum++
			btv.scanline = 0
			btv.extraScanlines = 0

			// record visible top/bottom for this frame
			btv.visibleTop = btv.thisVisibleTop
			btv.visibleBottom = btv.thisVisibleBottom

			// call new frame for all renderers
			for f := range btv.renderers {
				err := btv.renderers[f].NewFrame(btv.frameNum)
				if err != nil {
					return err
				}
			}

			// default top/bottom to the "ideal" values
			btv.thisVisibleTop = btv.spec.ScanlinesTotal
			btv.thisVisibleBottom = 0
		}

		btv.vsyncCount = 0
	}

	// push screen limits outwards as required
	if !sig.VBlank {
		if btv.scanline > btv.thisVisibleBottom {
			btv.thisVisibleBottom = btv.scanline

			// keep within limits
			if btv.thisVisibleBottom > btv.spec.ScanlinesTotal {
				btv.thisVisibleBottom = btv.spec.ScanlinesTotal
			}
		}
		if btv.scanline < btv.thisVisibleTop {
			btv.thisVisibleTop = btv.scanline
		}
	}

	// after the first frame, if there are "extra" scanlines then try changing
	// the tv specification.
	//
	// we are currently defining "extra" as 10. one extra scanline is too few.
	// for example, when using a value of one, the Fatal Run ROM experiences a
	// false change from NTSC to PAL between the resume/new screen and the game
	// "intro" screen. 10 is maybe too high but it's good for now.
	if btv.frameNum > 1 && btv.extraScanlines > 10 {
		_, err := btv.autoSpec()
		if err != nil {
			return err
		}
	}

	// record the current signal settings so they can be used for reference
	btv.prevSignal = sig

	// current coordinates
	x := int32(btv.horizPos) + int32(ClocksPerHblank)
	y := int32(btv.scanline)

	// decode color using the regular color signal
	red, green, blue := getColor(btv.spec, sig.Pixel)
	for f := range btv.renderers {
		err := btv.renderers[f].SetPixel(x, y, red, green, blue, sig.VBlank)
		if err != nil {
			return err
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

	// mix audio every other scanline
	if btv.scanline%2 == 0 {
		for f := range btv.mixers {
			err := btv.mixers[f].SetAudio(sig.Audio)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetState returns the TVState object for the named state. television
// implementations in other packages will difficulty extending this function
// because TVStateReq does not expose its members. (although it may need to if
// television is running in it's own goroutine)
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
	case ReqVisibleTop:
		return btv.visibleTop, nil
	case ReqVisibleBottom:
		return btv.visibleBottom, nil
	}
}

// GetSpec returns the television specification
func (btv StellaTelevision) GetSpec() *Specification {
	return btv.spec
}
