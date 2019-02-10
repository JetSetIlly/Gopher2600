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

	// if the most recently received signal is not as expected, according to
	// the television protocol definition in the Stella Programmer's Guide, the
	// the outOfSpec flags will be true
	outOfSpec bool

	// state of the television
	//	- the current horizontal position. the position where the next pixel will be
	//  drawn. also used to check we're receiving the correct signals at the
	//  correct time.
	horizPos TVState
	//	- the current frame
	frameNum TVState
	//	- the current scanline number
	scanline TVState

	// record of signal attributes from the last call to Signal()
	prevSignal SignalAttributes

	// vsyncCount records the number of consecutive colorClocks the vsync signal
	// has been sustained. we use this to help correctly implement vsync.
	vsyncCount int

	// the scanline at which the visible part of the screen begins and ends
	//	-- some ROMs make this more awkward than it first seems. for instance,
	//	vblank can be turned on and off anywhere, making detecting where the
	//	visible top of the screen more laborious than necessary.
	VisibleTop    int
	VisibleBottom int

	// callback hooks from Signal() - these are used by outer-structs to
	// hook into and add extra gubbins to the Signal() function
	HookNewFrame    func() error
	HookNewScanline func() error
	HookSetPixel    func(x, y int32, red, green, blue byte, vblank bool) error
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

	// initialise TVState
	tv.horizPos = TVState{label: "Horiz Pos", shortLabel: "HP", value: -tv.Spec.ClocksPerHblank, valueFormat: "%d"}
	tv.frameNum = TVState{label: "Frame", shortLabel: "FR", value: 0, valueFormat: "%d"}
	tv.scanline = TVState{label: "Scanline", shortLabel: "SL", value: 0, valueFormat: "%d"}

	tv.Reset()

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
	s := strings.Builder{}
	outOfSpec := ""
	if tv.outOfSpec {
		outOfSpec = " !!"
	}
	s.WriteString(fmt.Sprintf("TV (%s)%s:\n", tv.Spec.ID, outOfSpec))
	s.WriteString(fmt.Sprintf("   %s\n", tv.frameNum))
	s.WriteString(fmt.Sprintf("   %s\n", tv.scanline))
	s.WriteString(fmt.Sprintf("   %s", tv.horizPos))

	return s.String()
}

// map String to MachineInfo
func (tv HeadlessTV) String() string {
	return tv.MachineInfo()
}

// Reset all the values for the television
func (tv *HeadlessTV) Reset() error {
	tv.horizPos.value = -tv.Spec.ClocksPerHblank
	tv.frameNum.value = 0
	tv.scanline.value = 0
	tv.vsyncCount = 0
	tv.prevSignal = SignalAttributes{}

	tv.VisibleTop = -1
	tv.VisibleBottom = -1

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
func (tv *HeadlessTV) Signal(attr SignalAttributes) error {
	if attr.HSync && !tv.prevSignal.HSync {
		if tv.horizPos.value < -52 || tv.horizPos.value > -49 {
			panic(fmt.Sprintf("bad HSYNC (on at %d)", tv.horizPos.value))
		}
	} else if !attr.HSync && tv.prevSignal.HSync {
		if tv.horizPos.value < -36 || tv.horizPos.value > -33 {
			panic(fmt.Sprintf("bad HSYNC (off at %d)", tv.horizPos.value))
		}
	}
	if attr.CBurst && !tv.prevSignal.CBurst {
		if tv.horizPos.value < -28 || tv.horizPos.value > -17 {
			panic("bad CBURST (on)")
		}
	} else if !attr.CBurst && tv.prevSignal.CBurst {
		if tv.horizPos.value < -19 || tv.horizPos.value > -16 {
			panic("bad CBURST (off)")
		}
	}

	// simple implementation of vsync
	if attr.VSync {
		tv.vsyncCount++
	} else {
		if tv.vsyncCount >= tv.Spec.VsyncClocks {
			tv.outOfSpec = tv.vsyncCount != tv.Spec.VsyncClocks

			tv.frameNum.value++
			tv.scanline.value = 0
			tv.vsyncCount = 0

			err := tv.HookNewFrame()
			if err != nil {
				return err
			}

			tv.VisibleTop = -1
			tv.VisibleBottom = -1
		}
	}

	// start a new scanline if a frontporch signal has been received
	if attr.FrontPorch {
		tv.horizPos.value = -tv.Spec.ClocksPerHblank
		tv.scanline.value++
		err := tv.HookNewScanline()
		if err != nil {
			return err
		}

		if tv.scanline.value > tv.Spec.ScanlinesTotal {
			// we've not yet received a correct vsync signal
			// continue with an implied VSYNC
			tv.outOfSpec = true

			// repeat the last scanline (over and over if necessary)
			tv.scanline.value--
		}
	} else {
		tv.horizPos.value++

		// check to see if frontporch has been encountered
		// we're panicking because this shouldn't ever happen
		if tv.horizPos.value > tv.Spec.ClocksPerVisible {
			panic("no FRONTPORCH")
		}
	}

	// note the scanline when vblank is turned on/off. plus, only record the
	// off signal if it hasn't been set before during this frame
	if !attr.VBlank && tv.prevSignal.VBlank {
		if tv.VisibleTop == -1 {
			// some roms turn off vblank multiple times before the end of the frame.
			// if VisibleTop has been altered already then do not record the
			// VBlank off event
			//
			// ROMs affected:
			//	* Custer's Revenge
			//	* Ladybug
			tv.VisibleTop = tv.scanline.value
		}
	}
	if attr.VBlank && !tv.prevSignal.VBlank {
		// wierdly, some ROMS do not turn on VBlank until the beginning of a
		// frame.  this means that the value of vblank on will be less than
		// vblank off. the following condition prevents that.
		//
		// ROMs affected:
		//  * Gauntlet
		if tv.scanline.value == 0 {
			tv.VisibleBottom = tv.Spec.ScanlinesTotal
		} else {
			tv.VisibleBottom = tv.scanline.value
		}
	}

	// record the current signal settings so they can be used for reference
	tv.prevSignal = attr

	// decode color
	red, green, blue := byte(0), byte(0), byte(0)
	if attr.Pixel <= 256 {
		col := tv.Spec.Colors[attr.Pixel]
		red, green, blue = byte((col&0xff0000)>>16), byte((col&0xff00)>>8), byte(col&0xff)
	}

	// current coordinates
	x := int32(tv.horizPos.value) + int32(tv.Spec.ClocksPerHblank)
	y := int32(tv.scanline.value)

	return tv.HookSetPixel(x, y, red, green, blue, attr.VBlank)
}

// GetState returns the TVState object for the named state. television
// implementations in other packages will difficulty extending this function
// because TVStateReq does not expose its members. (although it may need to if
// television is running in it's own goroutine)
func (tv *HeadlessTV) GetState(request StateReq) (TVState, error) {
	switch request {
	default:
		return TVState{}, errors.NewGopherError(errors.UnknownTVRequest, request)
	case ReqFramenum:
		return tv.frameNum, nil
	case ReqScanline:
		return tv.scanline, nil
	case ReqHorizPos:
		return tv.horizPos, nil
	}
}

// GetMetaState returns the TVState object for the named state
func (tv *HeadlessTV) GetMetaState(request MetaStateReq) (string, error) {
	switch request {
	default:
		return "", errors.NewGopherError(errors.UnknownTVRequest, request)
	case ReqTVSpec:
		return tv.Spec.ID, nil
	}
}

// RegisterCallback is used to hook custom functionality into the televsion
func (tv *HeadlessTV) RegisterCallback(request CallbackReq, channel chan func(), callback func()) error {
	// the HeadlessTV implementation does nothing currently
	return errors.NewGopherError(errors.UnknownTVRequest, request)
}

// SetFeature is used to set a television attibute
func (tv *HeadlessTV) SetFeature(request FeatureReq, args ...interface{}) error {
	return errors.NewGopherError(errors.UnknownTVRequest, request)
}
