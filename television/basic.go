package television

import (
	"fmt"
	"gopher2600/debugger/metavideo"
	"gopher2600/errors"
	"strings"
)

// BasicTelevision is the minimalist implementation of the Television interface - a
// television without a screen. Fuller implementations of the television can
// use this as the basis of the emulation by struct embedding
type BasicTelevision struct {
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
}

// NewBasicTelevision creates a new instance of BasicTelevision for a minimalist
// implementation of a televsion for the VCS emulation
func NewBasicTelevision(tvType string) (*BasicTelevision, error) {
	btv := new(BasicTelevision)

	switch strings.ToUpper(tvType) {
	case "NTSC":
		btv.spec = SpecNTSC
	case "PAL":
		btv.spec = SpecPAL
	case "AUTO":
		btv.spec = SpecNTSC
		btv.auto = true
	default:
		return nil, errors.NewFormattedError(errors.BasicTelevision, fmt.Sprintf("unsupported tv type (%s)", tvType))
	}

	// initialise TVState
	btv.horizPos = -btv.spec.ClocksPerHblank
	btv.frameNum = 0
	btv.scanline = 0

	// empty list of renderers
	btv.renderers = make([]Renderer, 0)

	btv.Reset()

	return btv, nil
}

// MachineInfoTerse returns the television information in terse format
func (btv BasicTelevision) MachineInfoTerse() string {
	return fmt.Sprintf("FR=%d SL=%d HP=%d", btv.frameNum, btv.scanline, btv.horizPos)
}

// MachineInfo returns the television information in verbose format
func (btv BasicTelevision) MachineInfo() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("TV (%s)", btv.spec.ID))
	s.WriteString(fmt.Sprintf("\n   Frame: %d\n", btv.frameNum))
	s.WriteString(fmt.Sprintf("   Scanline: %d\n", btv.scanline))
	s.WriteString(fmt.Sprintf("   Horiz Pos: %d [%d]", btv.horizPos, btv.horizPos+btv.spec.ClocksPerHblank))

	return s.String()
}

// map String to MachineInfo
func (btv BasicTelevision) String() string {
	return btv.MachineInfo()
}

// AddRenderer adds a renderer implementation to the list
func (btv *BasicTelevision) AddRenderer(r Renderer) {
	btv.renderers = append(btv.renderers, r)
}

// Reset all the values for the television
func (btv *BasicTelevision) Reset() error {
	btv.horizPos = -btv.spec.ClocksPerHblank
	btv.frameNum = 0
	btv.scanline = 0
	btv.vsyncCount = 0
	btv.prevSignal = SignalAttributes{}

	// default top/bottom to the "ideal" values
	btv.thisVisibleTop = btv.spec.ScanlinesTotal
	btv.thisVisibleBottom = 0

	return nil
}

func (btv *BasicTelevision) autoSpec() (bool, error) {
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
func (btv *BasicTelevision) Signal(sig SignalAttributes) error {
	if sig.HSync && !btv.prevSignal.HSync {
		if btv.horizPos < -52 || btv.horizPos > -49 {
			return errors.NewFormattedError(errors.OutOfSpec, fmt.Sprintf("bad HSYNC (on at %d)", btv.horizPos))
		}
	} else if !sig.HSync && btv.prevSignal.HSync {
		if btv.horizPos < -36 || btv.horizPos > -33 {
			return errors.NewFormattedError(errors.OutOfSpec, fmt.Sprintf("bad HSYNC (off at %d)", btv.horizPos))
		}
	}
	if sig.CBurst && !btv.prevSignal.CBurst {
		if btv.horizPos < -28 || btv.horizPos > -17 {
			return errors.NewFormattedError(errors.OutOfSpec, fmt.Sprintf("bad CBURST (on at %d)", btv.horizPos))
		}
	} else if !sig.CBurst && btv.prevSignal.CBurst {
		if btv.horizPos < -19 || btv.horizPos > -16 {
			return errors.NewFormattedError(errors.OutOfSpec, fmt.Sprintf("bad CBURST (off at %d)", btv.horizPos))
		}
	}

	// start a new scanline if a frontporch signal has been received
	if sig.FrontPorch {
		btv.horizPos = -btv.spec.ClocksPerHblank

		// the frontporch signal implies the start of a new scanline
		btv.scanline++

		if btv.scanline > btv.spec.ScanlinesTotal {
			btv.scanline = btv.spec.ScanlinesTotal
		} else {
			for f := range btv.renderers {
				err := btv.renderers[f].NewScanline(btv.scanline)
				if err != nil {
					return err
				}
			}
		}
	} else {
		btv.horizPos++

		if btv.horizPos > btv.spec.ClocksPerVisible {
			return errors.NewFormattedError(errors.OutOfSpec, "no FRONTPORCH")
		}
	}

	// simple vsync implementation
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
			btv.thisVisibleBottom = btv.scanline + 2

			// keep within limits
			if btv.thisVisibleBottom > btv.spec.ScanlinesTotal {
				ok, err := btv.autoSpec()
				if err != nil {
					return err
				}
				if ok == false {
					btv.thisVisibleBottom = btv.spec.ScanlinesTotal
				}
			}
		}
		if btv.scanline < btv.thisVisibleTop {
			btv.thisVisibleTop = btv.scanline - 2

			// keep within limits
			if btv.thisVisibleTop < 0 {
				btv.thisVisibleTop = 0
			}
		}
	}

	// record the current signal settings so they can be used for reference
	btv.prevSignal = sig

	// current coordinates
	x := int32(btv.horizPos) + int32(btv.spec.ClocksPerHblank)
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
	red, green, blue = getColor(btv.spec, sig.AltPixel)
	for f := range btv.renderers {
		err := btv.renderers[f].SetAltPixel(x, y, red, green, blue, sig.VBlank)
		if err != nil {
			return err
		}
	}

	return nil
}

// MetaSignal recieves (and processes) additional emulator information from the emulator
func (btv *BasicTelevision) MetaSignal(metavideo.MetaSignalAttributes) error {
	return nil
}

// GetState returns the TVState object for the named state. television
// implementations in other packages will difficulty extending this function
// because TVStateReq does not expose its members. (although it may need to if
// television is running in it's own goroutine)
func (btv *BasicTelevision) GetState(request StateReq) (int, error) {
	switch request {
	default:
		return 0, errors.NewFormattedError(errors.UnknownTVRequest, request)
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
func (btv BasicTelevision) GetSpec() *Specification {
	return btv.spec
}
