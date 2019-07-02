package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"math/bits"
	"strings"
)

type scanCounter int

const scanCounterLimit scanCounter = 7

func (sc *scanCounter) start() {
	*sc = scanCounterLimit
}

func (sc scanCounter) active() bool {
	return sc >= 0 && sc <= scanCounterLimit
}

func (sc *scanCounter) tick() {
	*sc--
}

type playerSprite struct {
	// we need access to the TIA wide phase-clock and hsync polycounter in
	// order to ascertain both the reset position and current position of the
	// sprite in relation to the screen
	tiaClk *phaseclock.PhaseClock
	hsync  *polycounter.Polycounter

	// plus acces to the TIA wide delay "circuitry" when resetting sprite
	// position
	tiaDelay future.Scheduler

	// position of the sprite as a polycounter value - the basic principle
	// behind VCS sprites is to begin drawing of the sprite when position
	// circulates to zero
	//
	// why do we have an additional phaseclock (in addition to the TIA phase
	// clock that is)? from TIA_HW_Notes.txt:
	//
	// "Beside each counter there is a two-phase clock generator..."
	//
	// I've interpreted that to mean that each sprite has it's own phase clock
	// that can be reset and ticked indpendently. It seems to be correct.
	sprClk   phaseclock.PhaseClock
	position polycounter.Polycounter

	// in addition to the TIA-wide tiaDelay each sprite keeps track of its own
	// delays. this way, we can carefully control when the delayed sprite
	// events tick forwards - taking into consideration sprite specific
	// conditions
	SprDelay future.Ticker

	// horizontal movement
	moreHMOVE bool
	hmove     uint8

	// the following attributes are used for information purposes only
	//
	//  o the name of the sprite instance (eg. "player 0")
	//  o the pixel at which the sprite was reset
	//  o the pixel at which the sprite was reset plus any HMOVE modification
	//
	// see prepareForHMOVE() for a note on the presentation of hmovedPixel
	label       string
	resetPixel  int
	hmovedPixel int

	// ^^^ the above are common to all sprite types ^^^

	// player sprite attributes
	color         uint8
	size          uint8
	reflected     bool
	verticalDelay bool
	gfxDataNew    uint8
	gfxDataOld    uint8

	// scanCounter implements the "graphics scan counter" as described in
	// TIA_HW_Notes.txt:
	//
	// "The Player Graphics Scan Counters are 3-bit binary ripple counters
	// attached to the player objects, used to determine which pixel
	// of the player is currently being drawn by generating a 3-bit
	// source pixel address. These are the only binary ripple counters
	// in the TIA."
	scanCounter scanCounter

	// we need access to the other player sprite. when we write new gfxData, it
	// triggers the other player's gfxDataPrev value to equal the existing
	// gfxData of this player.
	//
	// this wasn't clear to me originally but was crystal clear after reading
	// Erik Mooney's post, "48-pixel highres routine explained!"
	otherPlayer *playerSprite

	// a record of the delayed start drawing event. resets to nil once drawing
	// commences
	startDrawingEvent *future.Event
}

func newPlayerSprite(label string, tiaclk *phaseclock.PhaseClock, hsync *polycounter.Polycounter, tiaDelay future.Scheduler) *playerSprite {

	ps := playerSprite{label: label, tiaClk: tiaclk, hsync: hsync, tiaDelay: tiaDelay}
	ps.position.SetLimit(39)
	ps.position.Reset()
	return &ps
}

// MachineInfo returns the player sprite information in terse format
func (ps playerSprite) MachineInfoTerse() string {
	return ps.String()
}

// MachineInfo returns the player sprite information in verbose format
func (ps playerSprite) MachineInfo() string {
	return ps.String()
}

func (ps playerSprite) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s %s [%03d ", ps.position, ps.sprClk, ps.resetPixel))
	s.WriteString(fmt.Sprintf("(%d)", int(ps.hmove)))
	s.WriteString(fmt.Sprintf(" %03d", ps.hmovedPixel))
	if ps.moreHMOVE && ps.hmove != 8 {
		s.WriteString("*]")
	} else {
		s.WriteString("]")
	}

	// notes

	if ps.moreHMOVE {
		s.WriteString(" hmoving")
	}

	if ps.scanCounter.active() {
		// add a comma if we've already noted something else
		if ps.moreHMOVE {
			s.WriteString(",")
		}

		s.WriteString(fmt.Sprintf(" drw (px %d)", ps.scanCounter))
	}

	return s.String()
}

// tick moves the counters (both position and graphics scan) along for the
// player sprite depending on whether HBLANK is active (visibleScreen) and the
// condition of the sprite's HMOVE counter
func (ps *playerSprite) tick(visibleScreen bool, hmoveCt uint8) {
	// check to see if there is more movement required for this sprite
	ps.moreHMOVE = ps.moreHMOVE && compareHMOVE(hmoveCt, ps.hmove)

	if visibleScreen || ps.moreHMOVE {
		// tick graphics scan counter during visible screen and during HMOVE.
		// from TIA_HW_Notes.txt:
		//
		// "Note that a HMOVE can gobble up the wrapped player graphics"
		//
		// in addition, the size value for the player affects how often the
		// scan counter ticks. from TIA_HW_Notes.txt:
		//
		// "The count frequency is determined by the NUSIZ register for that
		// player; this is used to selectively mask off the clock signals to
		// the Graphics Scan Counter. Depending on the player stretch mode,
		// one clock signal is allowed through every 1, 2 or 4 graphics CLK.
		// The stretched modes are derived from the two-phase clock; the H@2
		// phase allows 1 in 4 CLK through (4x stretch), both phases ORed
		// together allow 1 in 2 CLK through (2x stretch)."
		switch ps.size {
		case 0x05:
			if ps.sprClk.InPhase() || ps.sprClk.OutOfPhase() {
				ps.scanCounter.tick()
			}
		case 0x07:
			if ps.sprClk.InPhase() {
				ps.scanCounter.tick()
			}
		default:
			ps.scanCounter.tick()
		}

		// tick future events that are goverened by the sprite
		ps.SprDelay.Tick()

		// from TIA_HW_Notes.txt:
		//
		// "The [MOTCK] (motion clock?) line supplies the CLK signals
		// for all movable graphics objects during the visible part of
		// the scanline. It is an inverted (out of phase) CLK signal."
		ps.sprClk.Tick()
		if ps.sprClk.OutOfPhase() {
			// as per the comment above we only tick the position counter when the
			// sprite's clock is out of phase
			ps.position.Tick()

			// startDrawingEvent is delay by 5 ticks. from TIA_HW_Notes.txt:
			//
			// "Each START decode is delayed by 4 CLK in decoding, plus a
			// further 1 CLK to latch the graphics scan counter..."
			const startDelay = 5

			startDrawingEvent := func() {
				ps.scanCounter.start()
				ps.startDrawingEvent = nil
			}

			// "... The START decodes are ANDed with flags from the NUSIZ register
			// before being latched, to determine whether to draw that copy."
			switch ps.position.Count {
			case 3:
				if ps.size == 0x01 || ps.size == 0x03 {
					ps.startDrawingEvent = ps.SprDelay.Schedule(startDelay, startDrawingEvent, fmt.Sprintf("start drawing %s", ps.label))
				}
			case 7:
				if ps.size == 0x03 || ps.size == 0x02 || ps.size == 0x06 {
					ps.startDrawingEvent = ps.SprDelay.Schedule(startDelay, startDrawingEvent, fmt.Sprintf("start drawing %s", ps.label))
				}
			case 15:
				if ps.size == 0x04 || ps.size == 0x06 {
					ps.startDrawingEvent = ps.SprDelay.Schedule(startDelay, startDrawingEvent, fmt.Sprintf("start drawing %s", ps.label))
				}
			case 39:
				ps.startDrawingEvent = ps.SprDelay.Schedule(startDelay, startDrawingEvent, fmt.Sprintf("start drawing %s", ps.label))
			}
		}
	}
}

func (ps *playerSprite) prepareForHMOVE() {
	ps.moreHMOVE = true

	// adjust hmoved pixel now, with the caveat that the value is not valid
	// until the HMOVE has completed. presentation of this value should be
	// annotated suitably if HMOVE is in progress
	ps.hmovedPixel -= int(ps.hmove) - 8

	// adjust for screen boundary. silently ignoring values that are outside
	// the normal/expected range
	if ps.hmovedPixel < 0 {
		ps.hmovedPixel = ps.hmovedPixel + 160
	}
}

func (ps *playerSprite) resetPosition() {
	// delay of 5 clocks using tiaDelay rather than sprite delay. from
	// TIA_HW_Notes.txt:
	//
	// "This arrangement means that resetting the player counter on any
	// visible pixel will cause the main copy of the player to appear
	// at that same pixel position on the next and subsequent scanlines.
	// There are 5 CLK worth of clocking/latching to take into account,
	// so the actual position ends up 5 pixels to the right of the
	// reset pixel (approx. 9 pixels after the start of STA RESP0)."
	ps.SprDelay.Schedule(5, func() {
		// the pixel at which the sprite has been reset, in relation to the
		// left edge of the screen
		ps.resetPixel = (ps.hsync.Count * phaseclock.NumStates) + ps.tiaClk.Count()

		// adjust for screen boundaries
		ps.resetPixel -= 68
		if ps.resetPixel < -68 {
			d := ps.resetPixel + 68
			ps.resetPixel = 160 + d
		}

		// by definition the current pixel is the same as the reset pixel at
		// the moment of reset
		ps.hmovedPixel = ps.resetPixel

		// reset both sprite position and clock
		ps.position.Reset()
		ps.sprClk.Reset(false)

		// drop a running startDrawaingEvent from the delay queue
		if ps.startDrawingEvent != nil {
			ps.startDrawingEvent.Drop()
			ps.startDrawingEvent = nil
		}
	}, fmt.Sprintf("%s resetting position", ps.label))
}

// pixel returns the color of the player at the current time.  returns
// (false, col) if no pixel is to be seen; and (true, col) if there is
func (ps *playerSprite) pixel() (bool, uint8) {
	// select which graphics register to use
	gfxData := ps.gfxDataNew
	if ps.verticalDelay {
		gfxData = ps.gfxDataOld
	}

	// reverse the bits if necessary
	if ps.reflected {
		gfxData = bits.Reverse8(gfxData)
	}

	// pick the pixel from the gfxData register
	if ps.scanCounter.active() {
		if gfxData>>uint8(ps.scanCounter)&0x01 == 0x01 {
			return true, ps.color
		}
	}

	// always return player color because when in "scoremode" the playfield
	// wants to know the color of the player
	return false, ps.color
}

func (ps *playerSprite) setGfxData(data uint8) {
	// no delay necessary. from TIA_HW_Notes.txt:
	//
	// "Writes to GRP0 always modify the "new" P0 value, and the
	// contents of the "new" P0 are copied into "old" P0 whenever
	// GRP1 is written. (Likewise, writes to GRP1 always modify the
	// "new" P1 value, and the contents of the "new" P1 are copied
	// into "old" P1 whenever GRP0 is written). It is safe to modify
	// GRPn at any time, with immediate effect."
	ps.otherPlayer.gfxDataOld = ps.otherPlayer.gfxDataNew
	ps.gfxDataNew = data
}

func (ps *playerSprite) setVerticalDelay(vdelay bool) {
	// no delay necessary. from TIA_HW_Notes.txt:
	//
	// "Vertical Delay bit - this is also read every time a pixel is
	// generated and used to select which of the "new" (0) or "old" (1)
	// Player Graphics registers is used to generate the pixel. (ie
	// the pixel is retrieved from both registers in parallel, and
	// this flag used to choose between them at the graphics output).
	// It is safe to modify VDELPn at any time, with immediate effect."
	ps.verticalDelay = vdelay
}

func (ps *playerSprite) setHmoveValue(value uint8) {
	// horizontal movement values range from -8 to +7 for convenience we
	// convert this to the range 0 to 15. from TIA_HW_Notes.txt:
	//
	// "You may have noticed that the [...] discussion ignores the
	// fact that HMxx values are specified in the range +7 to -8.
	// In an odd twist, this was done purely for the convenience
	// of the programmer! The comparator for D7 in each HMxx latch
	// is wired up in reverse, costing nothing in silicon and
	// effectively inverting this bit so that the value can be
	// treated as a simple 0-15 count for movement left. It might
	// be easier to think of this as having D7 inverted when it
	// is stored in the first place."
	ps.hmove = (value ^ 0x80) >> 4
}

func (ps *playerSprite) setReflection(value bool) {
	// no delay necessary. from TIA_HW_Notes.txt:
	//
	// "Player Reflect bit - this is read every time a pixel is generated,
	// and used to conditionally invert the bits of the source pixel
	// address. This has the effect of flipping the player image drawn.
	// This flag could potentially be changed during the rendering of
	// the player, for example this might be used to draw bits 01233210."
	ps.reflected = value
}

func (ps *playerSprite) setNUSIZ(value uint8) {
	// no delay necessary. from TIA_HW_Notes.txt:
	//
	// "The NUSIZ register can be changed at any time in order to alter
	// the counting frequency, since it is read every graphics CLK.
	// This should allow possible player graphics warp effects etc."
	ps.size = value & 0x07
}

func (ps *playerSprite) setColor(value uint8) {
	// there is nothing in TIA_HW_Notes.txt about the color registers but I
	// don't believe there is a need for a delay
	ps.color = value
}
