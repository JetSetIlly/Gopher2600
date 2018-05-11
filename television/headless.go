package television

import (
	"fmt"
	"gopher2600/hardware/tia/video"
	"strings"
)

// HeadlessTV is the minimalist implementation of the Television interface - a
// television without a screen. fuller implementations of the television can
// use this as the basis of the emulation
type HeadlessTV struct {
	// spec is the specification of the tv type (NTSC or PAL)
	spec *specification

	// the current horizontal position. the position where the next pixel will be
	// drawn. also used to check we're receiving the correct signals at the
	// correct time.
	horizPos *TVState

	frameNum *TVState
	scanline *TVState

	// vsyncCount records the number of consecutive colorClocks the vsync signal
	// has been sustained
	vsyncCount int

	// records of signal information from the last call to Signal()
	prevHsync  bool
	prevCburst bool

	// if the signals we've received do not match what we expect then outOfSpec
	// will be false for the duration of the rest of the frame. this is useful
	// for ROM debugging, to indicate that the ROM may cause a real television to
	// misbehave.
	outOfSpec bool

	// callbacks
	newFrame    func() error
	newScanline func() error
	forceUpdate func() error
}

// NewHeadlessTV is the preferred method for initalising a headless TV
func NewHeadlessTV(tvType string) (*HeadlessTV, error) {
	tv := new(HeadlessTV)
	if tv == nil {
		return nil, fmt.Errorf("can't allocate memory for headless tv")
	}

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
		tv.spec = specNTSC
	case "PAL":
		tv.spec = specPAL
	default:
		return fmt.Errorf("unsupport tv type (%s)", tvType)
	}

	// empty callbacks
	tv.newFrame = func() error { return nil }
	tv.newScanline = func() error { return nil }
	tv.forceUpdate = func() error { return nil }

	// initialise TVState
	tv.horizPos = &TVState{label: "Horiz Pos", shortLabel: "HP", value: 0, valueFormat: "%d"}
	tv.frameNum = &TVState{label: "Frame", shortLabel: "FR", value: 0, valueFormat: "%d"}
	tv.scanline = &TVState{label: "Scanline", shortLabel: "SL", value: 0, valueFormat: "%d"}

	return nil
}

// StringTerse returns the television information in terse format
func (tv HeadlessTV) StringTerse() string {
	return fmt.Sprintf("%s %s %s", tv.frameNum.StringTerse(), tv.scanline.StringTerse(), tv.horizPos.StringTerse())
}

// String returns the television information in verbose format
func (tv HeadlessTV) String() string {
	return fmt.Sprintf("%v%v%v", tv.frameNum, tv.scanline, tv.horizPos)
}

// GetTVState returns the TVState object for the named state
func (tv HeadlessTV) GetTVState(state string) (*TVState, error) {
	switch state {
	default:
		return nil, fmt.Errorf("dummy tv doesn't have that tv state (%s)", state)
	case "FRAMENUM":
		return tv.frameNum, nil
	case "SCANLINE":
		return tv.scanline, nil
	case "HORIZPOS":
		return tv.horizPos, nil
	}
}

// ForceUpdate forces the tv image to be updated -- calls the forceUpdate
// callback from outside the television context (eg. from the debugger)
func (tv HeadlessTV) ForceUpdate() error {
	return tv.forceUpdate()
}

// Signal is principle method of communication between the VCS and televsion
func (tv *HeadlessTV) Signal(vsync, vblank, frontPorch, hsync, cburst bool, color video.Color) {

	// check that hsync signal is within the specification
	if hsync && !tv.prevHsync {
		if tv.horizPos.value < -52 || tv.horizPos.value > -49 {
			tv.outOfSpec = true
		}
	} else if !hsync && tv.prevHsync {
		if tv.horizPos.value < -36 || tv.horizPos.value > -33 {
			tv.outOfSpec = true
		}
	}

	// check that color burst signal is within the specification
	if cburst && !tv.prevCburst {
		if tv.horizPos.value < -28 || tv.horizPos.value > -17 {
			tv.outOfSpec = true
		}
	} else if !cburst && tv.prevCburst {
		if tv.horizPos.value < -19 || tv.horizPos.value > -16 {
			tv.outOfSpec = true
		}
	}

	// simple implementation of vsync
	if vsync {
		tv.vsyncCount++
	} else {
		if tv.vsyncCount > tv.spec.vsyncClocks {
			tv.outOfSpec = false
			tv.frameNum.value++
			tv.scanline.value = 0
			_ = tv.newFrame()
		}
		tv.vsyncCount = 0
	}

	// start a new scanline if a frontporch signal has been received
	if frontPorch {
		tv.horizPos.value = -tv.spec.clocksPerHblank
		tv.scanline.value++
		tv.newScanline()

		if tv.scanline.value > tv.spec.scanlinesTotal {
			// we've not yet received a correct vsync signal but we really should
			// have. continue but mark the frame as being out of spec
			tv.outOfSpec = true
		}
	} else {
		tv.horizPos.value++
		if tv.horizPos.value > tv.spec.clocksPerVisible {
			// we've not yet received a front porch signal yet but we really should
			// have. continue but mark the frame as being out of spec
			tv.outOfSpec = true
		}
	}

	// everthing else we could possibly do requires a screen of some sort
	// (eg. color decoding)

	// record the current signal settings so they can be used for reference
	tv.prevHsync = hsync
}

// SetVisibility does nothing for the HeadlessTV
func (tv HeadlessTV) SetVisibility(visible bool) error {
	return nil
}
