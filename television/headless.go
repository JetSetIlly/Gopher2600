package television

import (
	"fmt"
	"gopher2600/errors"
	"strings"
)

// HeadlessTV is the minimalist implementation of the Television interface - a
// television without a screen. Fuller implementations of the television can
// use this as the basis of the emulation by struct embedding. The
// InitHeadlessTV() method is useful in this regard.
type HeadlessTV struct {
	// spec is the specification of the tv type (NTSC or PAL)
	Spec *Specification

	// the current horizontal position. the position where the next pixel will be
	// drawn. also used to check we're receiving the correct signals at the
	// correct time.
	HorizPos *TVState

	// the current frame and scanline number
	FrameNum *TVState
	Scanline *TVState

	// record of signal attributes from the last call to Signal()
	prevSignal SignalAttributes

	// vsyncCount records the number of consecutive colorClocks the vsync signal
	// has been sustained. we use this to help correctly implement vsync.
	vsyncCount int

	// the scanline at which vblank is turned off and on
	//  - top mask ranges from 0 to VBlankOff-1
	//  - bottom mask ranges from VBlankOn to Spec.ScanlinesTotal
	VBlankOff int
	VBlankOn  int

	// if the signals we've received do not match what we expect then OutOfSpec
	// will be false for the duration of the rest of the frame. this is useful
	// for ROM debugging, to indicate that the ROM may cause a real television to
	// misbehave.
	OutOfSpec     bool
	OutOfSpecNote string

	// callback hooks from Signal()
	SignalNewFrameHook    func() error
	SignalNewScanlineHook func() error
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
		tv.Spec = SpecNTSC
	case "PAL":
		tv.Spec = SpecPAL
	default:
		return fmt.Errorf("unsupport tv type (%s)", tvType)
	}

	// empty callbacks
	tv.SignalNewFrameHook = func() error { return nil }
	tv.SignalNewScanlineHook = func() error { return nil }

	// initialise TVState
	tv.HorizPos = &TVState{label: "Horiz Pos", shortLabel: "HP", value: -tv.Spec.ClocksPerHblank, valueFormat: "%d"}
	tv.FrameNum = &TVState{label: "Frame", shortLabel: "FR", value: 0, valueFormat: "%d"}
	tv.Scanline = &TVState{label: "Scanline", shortLabel: "SL", value: 0, valueFormat: "%d"}

	// vblank off/on values
	tv.VBlankOff = 0
	tv.VBlankOn = tv.Spec.ScanlinesTotal

	return nil
}

// MachineInfoTerse returns the television information in terse format
func (tv HeadlessTV) MachineInfoTerse() string {
	specExclaim := ""
	if tv.OutOfSpec {
		specExclaim = " !!"
	}
	return fmt.Sprintf("%s %s %s%s", tv.FrameNum.MachineInfoTerse(), tv.Scanline.MachineInfoTerse(), tv.HorizPos.MachineInfoTerse(), specExclaim)
}

// MachineInfo returns the television information in verbose format
func (tv HeadlessTV) MachineInfo() string {
	s := strings.Builder{}
	outOfSpec := ""
	if tv.OutOfSpec {
		outOfSpec = " !!"
	}
	s.WriteString(fmt.Sprintf("TV (%s)%s:\n", tv.Spec.ID, outOfSpec))
	s.WriteString(fmt.Sprintf("   %s\n", tv.FrameNum))
	s.WriteString(fmt.Sprintf("   %s\n", tv.Scanline))
	s.WriteString(fmt.Sprintf("   %s", tv.HorizPos))

	return s.String()
}

// map String to MachineInfo
func (tv HeadlessTV) String() string {
	return tv.MachineInfo()
}

// Signal is principle method of communication between the VCS and televsion
func (tv *HeadlessTV) Signal(attr SignalAttributes) {

	// check that hsync signal is within the specification
	if attr.HSync && !tv.prevSignal.HSync {
		if tv.HorizPos.value < -52 || tv.HorizPos.value > -49 {
			tv.OutOfSpec = true
			tv.OutOfSpecNote = "bad HSYNC (on)"
		}
	} else if !attr.HSync && tv.prevSignal.HSync {
		if tv.HorizPos.value < -36 || tv.HorizPos.value > -33 {
			tv.OutOfSpec = true
			tv.OutOfSpecNote = "bad HSYNC (off)"
		}
	}

	// check that color burst signal is within the specification
	if attr.CBurst && !tv.prevSignal.CBurst {
		if tv.HorizPos.value < -28 || tv.HorizPos.value > -17 {
			tv.OutOfSpec = true
			tv.OutOfSpecNote = "bad CBURST (on)"
		}
	} else if !attr.CBurst && tv.prevSignal.CBurst {
		if tv.HorizPos.value < -19 || tv.HorizPos.value > -16 {
			tv.OutOfSpec = true
			tv.OutOfSpecNote = "bad CBURST (off)"
		}
	}

	// simple implementation of vsync
	if attr.VSync {
		tv.vsyncCount++
	} else {
		if tv.vsyncCount >= tv.Spec.VsyncClocks {
			tv.OutOfSpec = false
			tv.FrameNum.value++
			tv.Scanline.value = 0
			_ = tv.SignalNewFrameHook()
		}
		tv.vsyncCount = 0
	}

	// start a new scanline if a frontporch signal has been received
	if attr.FrontPorch {
		tv.HorizPos.value = -tv.Spec.ClocksPerHblank
		tv.Scanline.value++
		_ = tv.SignalNewScanlineHook()

		if tv.Scanline.value > tv.Spec.ScanlinesTotal {
			// we've not yet received a correct vsync signal but we really should
			// have. continue but mark the frame as being out of spec
			tv.OutOfSpec = true
			tv.OutOfSpecNote = "no VSYNC"
			tv.Scanline.value = 0
			_ = tv.SignalNewFrameHook()
		}
	} else {
		tv.HorizPos.value++
		if tv.HorizPos.value > tv.Spec.ClocksPerVisible {
			// we've not yet received a front porch signal yet but we really should
			// have. continue but mark the frame as being out of spec
			tv.OutOfSpec = true
			tv.OutOfSpecNote = "no FRONTPORCH"
		}
	}

	// note the scanline when vblank is turned on/off
	if !attr.VBlank && tv.prevSignal.VBlank {
		tv.VBlankOff = tv.Scanline.value
	}
	if attr.VBlank && !tv.prevSignal.VBlank {
		tv.VBlankOn = tv.Scanline.value
	}

	// record the current signal settings so they can be used for reference
	tv.prevSignal = attr

	// everthing else we could possibly do requires a screen of some sort
	// (eg. color decoding)
}

// RequestTVState returns the TVState object for the named state. television
// implementations in other packages will difficulty extending this function
// because TVStateReq does not expose its members.
func (tv *HeadlessTV) RequestTVState(request TVStateReq) (*TVState, error) {
	switch request {
	default:
		return nil, errors.NewGopherError(errors.UnknownTVRequest, request)
	case ReqFramenum:
		return tv.FrameNum, nil
	case ReqScanline:
		return tv.Scanline, nil
	case ReqHorizPos:
		return tv.HorizPos, nil
	}
}

// RequestTVInfo returns the TVState object for the named state
func (tv *HeadlessTV) RequestTVInfo(request TVInfoReq) (string, error) {
	switch request {
	default:
		return "", errors.NewGopherError(errors.UnknownTVRequest, request)
	case ReqTVSpec:
		return tv.Spec.ID, nil
	}
}

// RequestCallbackRegistration is used to hook custom functionality into the televsion
func (tv *HeadlessTV) RequestCallbackRegistration(request CallbackReq, channel chan func(), callback func()) error {
	// the HeadlessTV implementation does nothing currently
	return errors.NewGopherError(errors.UnknownTVRequest, request)
}

// RequestSetAttr is used to set a television attibute
func (tv *HeadlessTV) RequestSetAttr(request SetAttrReq, args ...interface{}) error {
	return errors.NewGopherError(errors.UnknownTVRequest, request)
}
