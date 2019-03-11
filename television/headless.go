package television

import (
	"fmt"
	"gopher2600/debugger/monitor"
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

	// if the most recently received signal is not as expected, according to
	// the television protocol definition in the Stella Programmer's Guide, the
	// the outOfSpec flags will be true
	outOfSpec bool

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

	// the scanline at which the visible part of the screen begins and ends
	// - we start off with ideal values and push the screen outwards as
	// required
	VisibleTop    int
	VisibleBottom int

	// pendingVisibleTop/Bottom records visible part of the screen (as
	// described above) during the frame. we use these to update the real
	// variables at the end of a frame
	pendingVisibleTop    int
	pendingVisibleBottom int

	// callback hooks from Signal() - these are used by outer-structs to
	// hook into and add extra gubbins to the Signal() function
	HookNewFrame    func() error
	HookNewScanline func() error
	HookSetPixel    func(x, y int32, red, green, blue byte, vblank bool) error
	HookSetAltPixel func(x, y int32, red, green, blue byte, vblank bool) error
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
	tv.HookNewFrame = func() error { return nil }
	tv.HookNewScanline = func() error { return nil }
	tv.HookSetPixel = func(x, y int32, r, g, b byte, vblank bool) error { return nil }
	tv.HookSetAltPixel = nil //func(x, y int32, r, g, b byte, vblank bool) error { return nil }

	// initialise TVState
	tv.horizPos = -tv.Spec.ClocksPerHblank
	tv.frameNum = 0
	tv.scanline = 0

	tv.Reset()

	return nil
}

// MachineInfoTerse returns the television information in terse format
func (tv HeadlessTV) MachineInfoTerse() string {
	specExclaim := ""
	if tv.outOfSpec {
		specExclaim = " !!"
	}
	return fmt.Sprintf("FR=%d SL=%d HP=%d %s", tv.frameNum, tv.scanline, tv.horizPos, specExclaim)
}

// MachineInfo returns the television information in verbose format
func (tv HeadlessTV) MachineInfo() string {
	s := strings.Builder{}
	outOfSpec := ""
	if tv.outOfSpec {
		outOfSpec = " !!"
	}
	s.WriteString(fmt.Sprintf("TV (%s)%s:\n", tv.Spec.ID, outOfSpec))
	s.WriteString(fmt.Sprintf("   Frame: %d\n", tv.frameNum))
	s.WriteString(fmt.Sprintf("   Scanline: %d\n", tv.scanline))
	s.WriteString(fmt.Sprintf("   Horiz Pos: %d [%d]", tv.horizPos, tv.horizPos+tv.Spec.ClocksPerHblank))

	return s.String()
}

// map String to MachineInfo
func (tv HeadlessTV) String() string {
	return tv.MachineInfo()
}

// Reset all the values for the television
func (tv *HeadlessTV) Reset() error {
	tv.horizPos = -tv.Spec.ClocksPerHblank
	tv.frameNum = 0
	tv.scanline = 0
	tv.vsyncCount = 0
	tv.prevSignal = SignalAttributes{}

	// default top/bottom to the "ideal" values
	tv.pendingVisibleTop = tv.Spec.IdealTop
	tv.pendingVisibleBottom = tv.Spec.IdealBottom

	return nil
}

// Signal is principle method of communication between the VCS and televsion
//
// the function will panic if an unexpected signal is received (or not received,
// as the case may be).
//
// if a signal is not entirely unexpected but is none-the-less out-of-spec then
// then the tv object will be flagged outOfSpec and the emulation allowed to
// continue.
func (tv *HeadlessTV) Signal(sig SignalAttributes) error {
	if sig.HSync && !tv.prevSignal.HSync {
		if tv.horizPos < -52 || tv.horizPos > -49 {
			//panic(fmt.Errorf("bad HSYNC (on at %d)", tv.horizPos))
			fmt.Println("hysnc on", tv.horizPos)
		}
	} else if !sig.HSync && tv.prevSignal.HSync {
		if tv.horizPos < -36 || tv.horizPos > -33 {
			//panic(fmt.Errorf("bad HSYNC (off at %d)", tv.horizPos))
			fmt.Println("hysnc off", tv.horizPos)
		}
	}
	if sig.CBurst && !tv.prevSignal.CBurst {
		if tv.horizPos < -28 || tv.horizPos > -17 {
			//panic(fmt.Errorf("bad CBURST (on)"))
			fmt.Println("cburst on", tv.horizPos)
		}
	} else if !sig.CBurst && tv.prevSignal.CBurst {
		if tv.horizPos < -19 || tv.horizPos > -16 {
			//panic(fmt.Errorf("bad CBURST (off)"))
			fmt.Println("cburst off", tv.horizPos)
		}
	}

	// simple implementation of vsync
	if sig.VSync {
		tv.vsyncCount++
	} else {
		if tv.vsyncCount >= tv.Spec.VsyncClocks {
			tv.outOfSpec = tv.vsyncCount != tv.Spec.VsyncClocks

			tv.frameNum++
			tv.scanline = 0
			tv.vsyncCount = 0

			// record visible top/bottom for this frame
			tv.VisibleTop = tv.pendingVisibleTop
			tv.VisibleBottom = tv.pendingVisibleBottom

			err := tv.HookNewFrame()
			if err != nil {
				return err
			}

			// default top/bottom to the "ideal" values
			tv.pendingVisibleTop = tv.Spec.IdealTop
			tv.pendingVisibleBottom = tv.Spec.IdealBottom
		}
	}

	// start a new scanline if a frontporch signal has been received
	if sig.FrontPorch {
		tv.horizPos = -tv.Spec.ClocksPerHblank
		tv.scanline++
		err := tv.HookNewScanline()
		if err != nil {
			return err
		}

		if tv.scanline > tv.Spec.ScanlinesTotal {
			// we've not yet received a correct vsync signal
			// continue with an implied VSYNC
			tv.outOfSpec = true

			// repeat the last scanline (over and over if necessary)
			tv.scanline--
		}
	} else {
		tv.horizPos++

		// check to see if frontporch has been encountered
		// we're panicking because this shouldn't ever happen
		if tv.horizPos > tv.Spec.ClocksPerVisible {
			panic(fmt.Errorf("no FRONTPORCH"))
		}
	}

	// push screen limits outwards as required
	if !sig.VBlank {
		if tv.scanline > tv.pendingVisibleBottom {
			tv.pendingVisibleBottom = tv.scanline + 2
		}
		if tv.scanline < tv.pendingVisibleTop {
			tv.pendingVisibleTop = tv.scanline - 2
		}
	}

	// record the current signal settings so they can be used for reference
	tv.prevSignal = sig

	// current coordinates
	x := int32(tv.horizPos) + int32(tv.Spec.ClocksPerHblank)
	y := int32(tv.scanline)

	// decode color using the regular color signal
	red, green, blue := tv.Spec.TranslateColorSignal(sig.Pixel)
	err := tv.HookSetPixel(x, y, red, green, blue, sig.VBlank)
	if err != nil {
		return err
	}

	// decode color using the alternative color signal
	if tv.HookSetAltPixel != nil {
		red, green, blue = tv.Spec.TranslateColorSignal(sig.AltPixel)
		return tv.HookSetAltPixel(x, y, red, green, blue, sig.VBlank)
	}

	return nil
}

// SystemStateRecord recieves (and processes) additional emulator information from the emulator
func (tv *HeadlessTV) SystemStateRecord(monitor.SystemState) error {
	return nil
}

// GetState returns the TVState object for the named state. television
// implementations in other packages will difficulty extending this function
// because TVStateReq does not expose its members. (although it may need to if
// television is running in it's own goroutine)
func (tv *HeadlessTV) GetState(request StateReq) (interface{}, error) {
	switch request {
	default:
		return nil, errors.NewGopherError(errors.UnknownTVRequest, request)
	case ReqFramenum:
		return tv.frameNum, nil
	case ReqScanline:
		return tv.scanline, nil
	case ReqHorizPos:
		return tv.horizPos, nil
	case ReqTVSpec:
		return tv.Spec.ID, nil
	}
}
