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

	// the colorClock shadows the colorClock of the VCS. the television doesn't
	// really have a colorClock but conceptually it is useful to think that it
	// does so we can check whether we're receiving the correct signal at the
	// correct time (to a tv engineer it would be more natural to measure time in
	// MHz)
	colorClock int

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

	// frame and scanline record information about the state of the current
	// image. this is not information that is integral to the televsion but is
	// useful for the VCS developer to know none-the-less.
	frame    int
	scanline int
}

// NewHeadlessTV is the preferred method for initalising a headless TV
func NewHeadlessTV(tvType string) (*HeadlessTV, error) {
	tv := new(HeadlessTV)
	if tv == nil {
		return nil, fmt.Errorf("can't allocate memory for headless tv")
	}

	switch strings.ToUpper(tvType) {
	case "NTSC":
		tv.spec = specNTSC
	case "PAL":
		tv.spec = specPAL
	default:
		return nil, fmt.Errorf("unsupport tv type (%s)", tvType)
	}

	return tv, nil
}

// StringTerse returns the television information in terse format
func (tv HeadlessTV) StringTerse() string {
	return fmt.Sprintf("F=%04d SL=%03d HP=%03d", tv.frame, tv.scanline, tv.colorClock-tv.spec.clocksPerHblank)
}

// String returns the television information in verbose format
func (tv HeadlessTV) String() string {
	return fmt.Sprintf("Frame: %04d\nScanline: %03d\nHoriz Pos: %03d\n", tv.frame, tv.scanline, tv.colorClock-tv.spec.clocksPerHblank)
}

// Signal is how the VCS communicates with the televsion
func (tv *HeadlessTV) Signal(vsync, vblank, frontPorch, hsync, cburst bool, color video.Color) {

	// check that hsync signal is within the specification
	if hsync && !tv.prevHsync {
		if tv.colorClock < 16 || tv.colorClock > 19 {
			tv.outOfSpec = true
		}
	} else if !hsync && tv.prevHsync {
		if tv.colorClock < 32 || tv.colorClock > 35 {
			tv.outOfSpec = true
		}
	}

	// check that color burst signal is within the specification
	if cburst && !tv.prevCburst {
		if tv.colorClock < 48 || tv.colorClock > 51 {
			tv.outOfSpec = true
		}
	} else if !cburst && tv.prevCburst {
		if tv.colorClock < 49 || tv.colorClock > 52 {
			tv.outOfSpec = true
		}
	}

	// simple implementation of vsync
	if vsync {
		tv.vsyncCount++
	} else {
		if tv.vsyncCount > tv.spec.vsyncClocks {
			tv.outOfSpec = false
			tv.frame++
			tv.scanline = 0
			// reset drawing routines
		}
		tv.vsyncCount = 0
	}

	// start a new scanline if a frontporch signal has been received
	if frontPorch {
		tv.colorClock = 0
		tv.scanline++

		if tv.scanline > tv.spec.scanlinesTotal {
			// we've not yet received a correct vsync signal but we really should
			// have. continue but mark the frame as being out of spec
			tv.outOfSpec = true
		}
	} else {
		tv.colorClock++
		if tv.colorClock > tv.spec.clocksPerScanline {
			// we've not yet received a front porch signal yet but we really should
			// have. continue but mark the frame as being out of spec
			tv.outOfSpec = true
		}
	}

	// TODO: color decoding

	// record the current signal settings so they can be used for reference
	tv.prevHsync = hsync
}
