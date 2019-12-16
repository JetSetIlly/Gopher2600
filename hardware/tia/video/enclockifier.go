package video

import (
	"fmt"
	"gopher2600/hardware/tia/future"
	"gopher2600/hardware/tia/phaseclock"
	"strings"
)

// enclockifier is the mechanism controlling how many pixels to output for both
// ball and missile sprites. it is the equivalent to the scanCounter mechanism
// used by the player sprite.
//
// the peculiar name is taken from TIA_HW_Notes:
//
// "Notes on the Ball/Missile width enclockifier
//
// Just to reiterate, ball width is given by combining clock signals
// of different widths based on the state of the two size bits (the
// gates form an AND -> OR -> AND -> OR -> out arrangement, with a
// hanger-on AND gate)."
//
// I've elected not to implement it exactly as above, preferring to use the
// future.Ticker mechanism used throughout the rest of the TIA emulation. see
// commentary in start() function for possible problems caused by this
// decision - start() rules may need some refinement.
type enclockifier struct {
	size  *uint8
	pclk  *phaseclock.PhaseClock
	delay *future.Ticker

	enable     bool
	secondHalf bool
	endEvent   *future.Event

	// which copy of the sprite is being drawn (ball sprite only ever has one
	// copy). value of zero means the primary copy is being drawn (if enable is
	// true)
	cpy int
}

func (en *enclockifier) String() string {
	s := strings.Builder{}
	if en.enable {
		if en.cpy > 0 {
			s.WriteString(fmt.Sprintf("+%d", en.cpy))
		}

		s.WriteString(fmt.Sprintf("(remaining %d", en.endEvent.RemainingCycles()))
		if en.secondHalf {
			s.WriteString("/2nd")
		}
		s.WriteString(")")
	}
	return s.String()
}

// the ball sprite drops enclockifier events during position resets
func (en *enclockifier) drop() {
	if en.endEvent != nil {
		en.enable = false
		en.endEvent.Drop()
		en.endEvent = nil
	}
}

// the ball sprite forces conclusion (or continuation in the case of 8x widht)
// of enclockifier events during position resets
func (en *enclockifier) force() {
	if en.endEvent != nil {
		en.endEvent.Force()
	}
}

// pause end event. there's no need for a corresponding resume() function
func (en *enclockifier) pause() {
	if en.endEvent != nil {
		en.endEvent.Pause()
	}
}

func (en *enclockifier) aboutToEnd() bool {
	if en.endEvent == nil {
		return false
	}
	return en.endEvent.AboutToEnd()
}

func (en *enclockifier) start() {
	en.enable = true

	// upon receiving a start signal, we decide for how long the enable flag
	// should be true. after the requisite number of clocks endEvent() is run,
	// disabling the flag.
	//
	// what's not clear as yet is what happens if the size value of the sprite
	// changes while the enable flag is true. indeed I'm not sure if this is
	// acutally possible. if it is, then we may need to refine how we do all of
	// this.

	switch *en.size {
	case 0x00:
		en.endEvent = en.delay.Schedule(1, en._futureOnEnd, "END")
	case 0x01:
		en.endEvent = en.delay.Schedule(2, en._futureOnEnd, "END")
	case 0x02:
		en.endEvent = en.delay.Schedule(4, en._futureOnEnd, "END")
	case 0x03:
		// from TIA_HW_Notes.txt:
		//
		// "The second half is added if both D4 and D5 are set; a delayed copy
		// of the Start signal (4 colour CLK wide again) is OR-ed into the
		// Enable signal at the final OR gate."
		en.endEvent = en.delay.Schedule(4, en._futureOnEndSecond, "END (1st half)")
	}
}

// called at very end of enclockifier sequence
func (en *enclockifier) _futureOnEnd() {
	en.enable = false
	en.endEvent = nil
	en.secondHalf = false
}

// called at end of enclockifier sequence for quadruple width sprites. calls
// _futureOnEnd at end of second half
func (en *enclockifier) _futureOnEndSecond() {
	en.secondHalf = true
	en.endEvent = en.delay.Schedule(4, en._futureOnEnd, "END (2nd half)")
}
