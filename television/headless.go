package television

import (
	"fmt"
	"gopher2600/errors"
	"strings"
)

// list of valid tv state requests for headless tv
const (
	ReqFramenum TVStateReq = "FRAME"
	ReqScanline TVStateReq = "SCANLINE"
	ReqHorizPos TVStateReq = "HORIZPOS"
)

// HeadlessTV is the minimalist implementation of the Television interface - a
// television without a screen. Fuller implementations of the television can
// use this as the basis of the emulation by struct embedding. The
// InitHeadlessTV() method is useful in this regard.
type HeadlessTV struct {
	// spec is the specification of the tv type (NTSC or PAL)
	Spec *specification

	// the current horizontal position. the position where the next pixel will be
	// drawn. also used to check we're receiving the correct signals at the
	// correct time.
	horizPos *TVState

	// the current frame and scanline number
	frameNum *TVState
	scanline *TVState

	// vsyncCount records the number of consecutive colorClocks the vsync signal
	// has been sustained
	vsyncCount int

	// records of signal information from the last call to Signal()
	prevHSync  bool
	prevCBurst bool

	// if the signals we've received do not match what we expect then outOfSpec
	// will be false for the duration of the rest of the frame. this is useful
	// for ROM debugging, to indicate that the ROM may cause a real television to
	// misbehave.
	outOfSpec bool

	// phospher indicates whether the phosphor gun is active
	Phosphor bool

	// callbacks
	NewFrame    func() error
	NewScanline func() error
	forceUpdate func() error
}

// NewHeadlessTV creates a new instance of HeadlessTV for a minimalist
// implementation of a televsion for the VCS emulation
func NewHeadlessTV(tvType string) (*HeadlessTV, error) {
	tv := new(HeadlessTV)

	err := InitHeadlessTV(tv, tvType)
	if err != nil {
		return nil, err
	}

	return tv, nil
}

// InitHeadlessTV initialises an instance of HeadlessTV. useful for television
// types that want to "inherit" the basic functionality of HeadlessTV by
// embedding. those types can call InitHeadlessTV() on the embedded field
func InitHeadlessTV(tv *HeadlessTV, tvType string) error {
	switch strings.ToUpper(tvType) {
	case "NTSC":
		tv.Spec = specNTSC
	case "PAL":
		tv.Spec = specPAL
	default:
		return fmt.Errorf("unsupport tv type (%s)", tvType)
	}

	// empty callbacks
	tv.NewFrame = func() error { return nil }
	tv.NewScanline = func() error { return nil }
	tv.forceUpdate = func() error { return nil }

	// initialise TVState
	tv.horizPos = &TVState{label: "Horiz Pos", shortLabel: "HP", value: -tv.Spec.ClocksPerHblank, valueFormat: "%d"}
	tv.frameNum = &TVState{label: "Frame", shortLabel: "FR", value: 0, valueFormat: "%d"}
	tv.scanline = &TVState{label: "Scanline", shortLabel: "SL", value: 0, valueFormat: "%d"}

	return nil
}

// MachineInfoTerse returns the television information in terse format
func (tv HeadlessTV) MachineInfoTerse() string {
	specExclaim := ""
	if tv.outOfSpec {
		specExclaim = " !!"
	}
	return fmt.Sprintf("%s %s %s%s", tv.frameNum.MachineInfoTerse(), tv.scanline.MachineInfoTerse(), tv.horizPos.MachineInfoTerse(), specExclaim)
}

// MachineInfo returns the television information in verbose format
func (tv HeadlessTV) MachineInfo() string {
	outOfSpec := ""
	if tv.outOfSpec {
		outOfSpec = "!!"
	}
	return fmt.Sprintf("%v\n%v\n%v%s\nPixel: %d", tv.frameNum, tv.scanline, tv.horizPos, outOfSpec, tv.PixelX(false))
}

// map String to MachineInfo
func (tv HeadlessTV) String() string {
	return tv.MachineInfo()
}

// PixelX returns an adjusted horizPos value
// -- adjustOrigin argument specifies whether or not pixel origin should be the
// visible portion of the screen
// -- note that if adjust origin is true, the function may return a negative
// number
func (tv HeadlessTV) PixelX(adjustOrigin bool) int {
	if adjustOrigin {
		return tv.horizPos.value
	}
	return tv.horizPos.value + tv.Spec.ClocksPerHblank
}

// PixelY returns an adjusted scanline value
// -- adjustOrigin argument specifies whether or not pixel origin should be the
// visible portion of the screen
// -- note that if adjust origin is true, the function may return a negative
// number
func (tv HeadlessTV) PixelY(adjustOrigin bool) int {
	if adjustOrigin {
		return tv.scanline.value - tv.Spec.ScanlinesPerVBlank
	}
	return tv.scanline.value
}

// ForceUpdate forces the tv image to be updated -- calls the forceUpdate
// callback from outside the television context (eg. from the debugger)
func (tv HeadlessTV) ForceUpdate() error {
	return tv.forceUpdate()
}

// Signal is principle method of communication between the VCS and televsion
func (tv *HeadlessTV) Signal(attr SignalAttributes) {

	// check that hsync signal is within the specification
	if attr.HSync && !tv.prevHSync {
		if tv.horizPos.value < -52 || tv.horizPos.value > -49 {
			tv.outOfSpec = true
		}
	} else if !attr.HSync && tv.prevHSync {
		if tv.horizPos.value < -36 || tv.horizPos.value > -33 {
			tv.outOfSpec = true
		}
	}

	// check that color burst signal is within the specification
	if attr.CBurst && !tv.prevCBurst {
		if tv.horizPos.value < -28 || tv.horizPos.value > -17 {
			tv.outOfSpec = true
		}
	} else if !attr.CBurst && tv.prevCBurst {
		if tv.horizPos.value < -19 || tv.horizPos.value > -16 {
			tv.outOfSpec = true
		}
	}

	// simple implementation of vsync
	if attr.VSync {
		tv.vsyncCount++
	} else {
		if tv.vsyncCount >= tv.Spec.VsyncClocks {
			tv.outOfSpec = false
			tv.frameNum.value++
			tv.scanline.value = 0
			_ = tv.NewFrame()
		}
		tv.vsyncCount = 0
	}

	// start a new scanline if a frontporch signal has been received
	if attr.FrontPorch {
		tv.horizPos.value = -tv.Spec.ClocksPerHblank
		tv.scanline.value++
		tv.NewScanline()

		if tv.scanline.value > tv.Spec.ScanlinesTotal {
			// we've not yet received a correct vsync signal but we really should
			// have. continue but mark the frame as being out of spec
			tv.outOfSpec = true
		}
	} else {
		tv.horizPos.value++
		if tv.horizPos.value > tv.Spec.ClocksPerVisible {
			// we've not yet received a front porch signal yet but we really should
			// have. continue but mark the frame as being out of spec
			tv.outOfSpec = true
		}
	}

	// set phosphor state
	tv.Phosphor = tv.horizPos.value >= 0 && !attr.VBlank

	// record the current signal settings so they can be used for reference
	tv.prevHSync = attr.HSync

	// everthing else we could possibly do requires a screen of some sort
	// (eg. color decoding)
}

// SetVisibility does nothing for the HeadlessTV
func (tv *HeadlessTV) SetVisibility(visible bool) error {
	return nil
}

// SetPause does nothing for the HeadlessTV
func (tv *HeadlessTV) SetPause(pause bool) error {
	return nil
}

// RequestTVState returns the TVState object for the named state
func (tv *HeadlessTV) RequestTVState(request TVStateReq) (*TVState, error) {
	switch request {
	default:
		return nil, errors.NewGopherError(errors.UnknownStateRequest, request)
	case ReqFramenum:
		return tv.frameNum, nil
	case ReqScanline:
		return tv.scanline, nil
	case ReqHorizPos:
		return tv.horizPos, nil
	}
}

// RegisterCallback (with dummyTV reciever) is the null implementation
func (tv *HeadlessTV) RegisterCallback(request CallbackReq, callback func()) error {
	return errors.NewGopherError(errors.UnknownCallbackRequest, request)
}
