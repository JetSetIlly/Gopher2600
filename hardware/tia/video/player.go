package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/television"
	"math/bits"
	"strings"
)

type scanCounter struct {
	offset  int
	latches int
}

const scanCounterLimit int = 7

func (sc *scanCounter) start(nusiz uint8) {
	if nusiz == 0x05 || nusiz == 0x07 {
		sc.latches = 2
	} else {
		sc.latches = 1
	}
}

func (sc scanCounter) active() bool {
	return sc.offset >= 0 && sc.offset <= scanCounterLimit
}

func (sc scanCounter) isLatching() bool {
	return sc.latches > 0
}

func (sc *scanCounter) tick() {
	if sc.latches > 0 {
		sc.latches--
		if sc.latches == 0 {
			sc.offset = scanCounterLimit
		}
	} else if sc.offset >= 0 {
		sc.offset--
	}
}

type playerSprite struct {
	// we need a reference to the attached television so that we can note the
	// reset position of the sprite
	//
	// should we rely on the television implementation to report this
	// information? I think so. the purpose of noting the reset position at all
	// is so that we can debug (both the emulator and any games we're
	// developing) more easily. if we calculate the reset position another way,
	// using only information from the TIA, there's a risk that the debugging
	// information from the TV and from the sprite will differ - to the point
	// of confusion.
	tv television.Television

	// references to some fundamental TIA properties. various combinations of
	// these affect the latching delay when resetting the sprite
	hblank     *bool
	hmoveLatch *bool

	// ^^^ references to other parts of the VCS ^^^

	// position of the sprite as a polycounter value - the basic principle
	// behind VCS sprites is to begin drawing the sprite when position
	// circulates to zero
	position polycounter.Polycounter

	// "Beside each counter there is a two-phase clock generator..."
	pclk phaseclock.PhaseClock

	// in addition to the TIA-wide tiaDelay each sprite keeps track of its own
	// delays. this way, we can carefully control when the sprite events occur
	// - taking into consideration sprite specific conditions
	//
	// sprites mainly use their own delay but some operations require the
	// TIA-wide delay. for those instances a future.Scheduler instance is
	// passed to the required function
	Delay future.Ticker

	// horizontal movement
	moreHMOVE bool
	hmove     uint8

	// the following attributes are used for information purposes only:

	// the name of the sprite instance (eg. "player 0")
	label string

	// the pixel at which the sprite was reset. in the case of the ball and
	// missile sprites the scan counter starts at the resetPixel. for the
	// player sprite however, there is additional latching to consider. rather
	// than introducing an additional variable keeping track of the start
	// pixel, the resetPixel is modified according to the player sprite's
	// current NUSIZ.
	resetPixel int

	// the pixel at which the sprite was reset plus any HMOVE modification see
	// prepareForHMOVE() for a note on the presentation of hmovedPixel
	hmovedPixel int

	// the number of times the sprite has been ticked since last reset or since
	// the position polycounter cycled
	//
	// cpu cycles can be attained by dividing numTicks by 3
	videoCycles int

	// ^^^ the above are common to all sprite types ^^^

	// player sprite attributes
	color         uint8
	nusiz         uint8
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

func newPlayerSprite(label string, tv television.Television, hblank, hmoveLatch *bool) *playerSprite {
	ps := playerSprite{
		label:      label,
		tv:         tv,
		hblank:     hblank,
		hmoveLatch: hmoveLatch,
	}
	ps.position.Reset()
	return &ps
}

// MachineInfo returns the player sprite information in terse format
func (ps playerSprite) MachineInfoTerse() string {
	return ps.String()
}

// MachineInfo returns the player sprite information in verbose format
func (ps playerSprite) MachineInfo() string {
	s := strings.Builder{}
	s.WriteString(ps.String())
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("gfx new: %08b", ps.gfxDataNew))
	if !ps.verticalDelay {
		s.WriteString(" *")
	}
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("gfx old: %08b", ps.gfxDataOld))
	if ps.verticalDelay {
		s.WriteString(" *")
	}
	return s.String()
}

func (ps playerSprite) String() string {
	// the hmove value as maintained by the sprite type is normalised for
	// for purposes of presentation. put the sign bit back to reflect the
	// original value as used in the ROM.
	normalisedHmove := int(ps.hmove) | 0x08

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s %s [%03d ", ps.position, ps.pclk, ps.resetPixel))
	s.WriteString(fmt.Sprintf("(%1x)", normalisedHmove))
	s.WriteString(fmt.Sprintf(" %03d", ps.hmovedPixel))
	if ps.moreHMOVE && ps.hmove != 8 {
		s.WriteString("*] ")
	} else {
		s.WriteString("] ")
	}

	// interpret nusiz value
	switch ps.nusiz {
	case 0x0:
		s.WriteString("|")
	case 0x1:
		s.WriteString("|_|")
	case 0x2:
		s.WriteString("|__|")
	case 0x3:
		s.WriteString("|_|_|")
	case 0x4:
		s.WriteString("|___|")
	case 0x5:
		s.WriteString("||")
	case 0x6:
		s.WriteString("|__|__|")
	case 0x7:
		s.WriteString("||||")
	}

	// notes
	extra := false

	if ps.moreHMOVE {
		s.WriteString(" hmoving")
		extra = true
	}

	if ps.scanCounter.active() {
		// add a comma if we've already noted something else
		if extra {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw (px %d)", ps.scanCounter.offset))
		extra = true
	} else if ps.scanCounter.isLatching() {
		// add a comma if we've already noted something else
		if extra {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw (in %d)", ps.scanCounter.latches))
		extra = true
	}

	if ps.verticalDelay {
		if extra {
			s.WriteString(",")
		}
		s.WriteString(" vdel")
		extra = true
	}

	if ps.reflected {
		if extra {
			s.WriteString(",")
		}
		s.WriteString(" ref")
		// extra = true
	}

	return s.String()
}

// tick moves the sprite counters along (both position and graphics scan).
//
// note that the extra clock value caused by an active HMOVE, is not supplied
// directly.  that the existance of the extra clock is derived in this tick
// function, depending on the supplied hmoveCt and the whether the sprite's own
// HMOVE value suggests that there should be more movement. see compareHMOVE()
// for details
func (ps *playerSprite) tick(motck bool, hmove bool, hmoveCt uint8) {
	// check to see if there is more movement required for this sprite
	if hmove {
		ps.moreHMOVE = ps.moreHMOVE && compareHMOVE(hmoveCt, ps.hmove)
	}

	if (hmove && ps.moreHMOVE) || motck {
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
		switch ps.nusiz {
		case 0x05:
			if ps.pclk.LatePhi2() || ps.pclk.LatePhi1() || ps.scanCounter.isLatching() {
				ps.scanCounter.tick()
			}
		case 0x07:
			if ps.pclk.LatePhi2() || ps.scanCounter.isLatching() {
				ps.scanCounter.tick()
			}
		default:
			ps.scanCounter.tick()
		}

		ps.pclk.Tick()

		// I cannot find a direct reference that describes when position
		// counters are ticked forward. however, TIA_HW_Notes.txt does say the
		// HSYNC counter ticks forward on the rising edge of Phi2. it is
		// reasonable to assume that the sprite position counters do likewise.
		if ps.pclk.Phi2() {
			if ps.position.Tick() {
				ps.videoCycles = 0
			} else {
				ps.videoCycles++
			}

			// startDrawingEvent is delayed by 5 ticks. from TIA_HW_Notes.txt:
			//
			// "Each START decode is delayed by 4 CLK in decoding, plus a
			// further 1 CLK to latch the graphics scan counter..."
			//
			// the "further 1 CLK" is actually a further 2 CLKs in the case of
			// 2x and 4x size sprites. we'll handle the additional latching in
			// the scan counter
			//
			// note that the additional latching has an impact of what we
			// report as being the reset pixel.
			const startDelay = 4

			startDrawingEvent := func() {
				ps.startDrawingEvent = nil
				ps.scanCounter.start(ps.nusiz)
			}

			// "... The START decodes are ANDed with flags from the NUSIZ register
			// before being latched, to determine whether to draw that copy."
			switch ps.position.Count {
			case 3:
				if ps.nusiz == 0x01 || ps.nusiz == 0x03 {
					ps.startDrawingEvent = ps.Delay.Schedule(startDelay, startDrawingEvent, "START")
				}
			case 7:
				if ps.nusiz == 0x03 || ps.nusiz == 0x02 || ps.nusiz == 0x06 {
					ps.startDrawingEvent = ps.Delay.Schedule(startDelay, startDrawingEvent, "START")
				}
			case 15:
				if ps.nusiz == 0x04 || ps.nusiz == 0x06 {
					ps.startDrawingEvent = ps.Delay.Schedule(startDelay, startDrawingEvent, "START")
				}
			case 39:
				ps.startDrawingEvent = ps.Delay.Schedule(startDelay, startDrawingEvent, "START")

			case 40:
				ps.position.Reset()
			}
		}

		// tick future events that are goverened by the sprite
		ps.Delay.Tick()
	}
}

func (ps *playerSprite) prepareForHMOVE() {
	ps.moreHMOVE = true

	// adjust hmoved pixel now, with the caveat that the value is not valid
	// until the HMOVE has completed. in the MachineInfo() function this value
	// is annotated with a "*" to indicate that HMOVE is still in progress
	ps.hmovedPixel -= int(ps.hmove) - 8

	// adjust for screen boundary. silently ignoring values that are outside
	// the normal/expected range
	if ps.hmovedPixel < 0 {
		ps.hmovedPixel += ps.tv.GetSpec().ClocksPerVisible
	}
}

func (ps *playerSprite) resetPosition() {
	// delay of 5 clocks using. from TIA_HW_Notes.txt:
	//
	// "This arrangement means that resetting the player counter on any
	// visible pixel will cause the main copy of the player to appear
	// at that same pixel position on the next and subsequent scanlines.
	// There are 5 CLK worth of clocking/latching to take into account,
	// so the actual position ends up 5 pixels to the right of the
	// reset pixel (approx. 9 pixels after the start of STA RESP0)."
	delay := 4

	// if we're scheduling the reset during a HBLANK however there are extra
	// conditions which adjust the delay value. these figures have been gleaned
	// through observation. with some supporting notes from the following
	// thread.
	//
	// https://atariage.com/forums/topic/207444-questionproblem-about-sprite-positioning-during-hblank/
	//
	// that said, I'm not entirely sure what's going on and why these
	// adjustments are required.
	if *ps.hblank {
		if *ps.hmoveLatch {
			delay = 3
		} else {
			delay = 2
		}
	}

	// pause pending start drawing events
	if ps.startDrawingEvent != nil {
		ps.startDrawingEvent.Pause()
	}

	ps.Delay.Schedule(delay, func() {
		// the pixel at which the sprite has been reset, in relation to the
		// left edge of the screen
		ps.resetPixel, _ = ps.tv.GetState(television.ReqHorizPos)

		// resetPixel adjusted because the tv is not yet at the position of the
		// new pixel (+1) and another +1 because of the additional clock
		// for player sprites after the start signal
		ps.resetPixel += 2

		// if size is 2x or 4x then we need an additional reset pixel
		//
		// note that we need to monkey with resetPixel whenever NUSIZ changes.
		// see setNUSIZ() function below
		if ps.nusiz == 0x05 || ps.nusiz == 0x07 {
			ps.resetPixel++
		}

		// adjust resetPixel for screen boundaries
		if ps.resetPixel > ps.tv.GetSpec().ClocksPerVisible {
			ps.resetPixel -= ps.tv.GetSpec().ClocksPerVisible
		}

		// by definition the current pixel is the same as the reset pixel at
		// the moment of reset
		ps.hmovedPixel = ps.resetPixel

		// reset both sprite position and clock
		ps.position.Reset()
		ps.pclk.Reset()

		// a player reset doesn't normally start drawing straight away unless
		// one was a about to start (within 2 cycles from when the reset was first
		// triggered)
		//
		// if a pending drawing event was more than two cycles away it is
		// dropped
		//
		// rules discovered through observation
		if ps.startDrawingEvent != nil {
			if ps.startDrawingEvent.RemainingCycles <= 2 {
				ps.startDrawingEvent.Force()
			} else {
				ps.startDrawingEvent.Drop()
				ps.startDrawingEvent = nil
			}
		}

		// reset cycle counter
		ps.videoCycles = 0
	}, "RESPx")
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
		if gfxData>>uint8(ps.scanCounter.offset)&0x01 == 0x01 {
			return true, ps.color
		}
	}

	// always return player color because when in "scoremode" the playfield
	// wants to know the color of the player
	return false, ps.color
}

func (ps *playerSprite) setGfxData(data uint8) {
	// from TIA_HW_Notes.txt:
	//
	// "Writes to GRP0 always modify the "new" P0 value, and the contents of
	// the "new" P0 are copied into "old" P0 whenever GRP1 is written.
	// (Likewise, writes to GRP1 always modify the "new" P1 value, and the
	// contents of the "new" P1 are copied into "old" P1 whenever GRP0 is
	// written). It is safe to modify GRPn at any time, with immediate effect."
	//
	// observation suggests that rather than being safe to "modify GRPn at any
	// time", a delay of 1 cycle is required.
	//
	// * Barnstormer scanline 61 demonstrates perfectly why we need this delay
	// * the value of 1 is by no means certain. it could be greater than 1
	//
	// !!TODO: more work on player.setGfxData()
	ps.Delay.Schedule(1, func() {
		ps.otherPlayer.gfxDataOld = ps.otherPlayer.gfxDataNew
		ps.gfxDataNew = data
	}, "GRPx")
}

func (ps *playerSprite) setVerticalDelay(vdelay bool) {
	// from TIA_HW_Notes.txt:
	//
	// "Vertical Delay bit - this is also read every time a pixel is generated
	// and used to select which of the "new" (0) or "old" (1) Player Graphics
	// registers is used to generate the pixel. (ie the pixel is retrieved from
	// both registers in parallel, and this flag used to choose between them at
	// the graphics output).  It is safe to modify VDELPn at any time, with
	// immediate effect."
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
	// from TIA_HW_Notes.txt:
	//
	// "Player Reflect bit - this is read every time a pixel is generated,
	// and used to conditionally invert the bits of the source pixel
	// address. This has the effect of flipping the player image drawn.
	// This flag could potentially be changed during the rendering of
	// the player, for example this might be used to draw bits 01233210."
	ps.reflected = value
}

func (ps *playerSprite) setNUSIZ(value uint8) {
	// from TIA_HW_Notes.txt:
	//
	// "The NUSIZ register can be changed at any time in order to alter
	// the counting frequency, since it is read every graphics CLK.
	// This should allow possible player graphics warp effects etc."

	// whilst the notes say that the register can be changed at any time, there
	// is a delay of sorts in certain situations:
	var delay int

	if ps.startDrawingEvent != nil {
		// if the sprite is scheduled to start drawing the delay is equal to the
		// number of pre-draw-latches that are required depending on the current
		// size.
		if ps.nusiz == 0x05 || ps.nusiz == 0x07 {
			delay = 2
		} else {
			delay = 1
		}
	} else if ps.scanCounter.active() {
		// if the sprite is currently in its draw sequence (ie. the scan
		// counter is active) then the delay depends on waiting for the current
		// "pixel" to have been completely drawn. This depends on the current
		// value of NUSIZ and the current state of the phase clock.
		delay = 2
	} else {
		// if the there is no scheduled drawing event or one currenty taking place,
		// then the dealy is equal to the number of latches currently outstanding.
		// in this instance, if the drawing is actually running then the number of
		// latches, and therefore the delay, will be zero.
		delay = ps.scanCounter.latches
	}

	// note that this delay does not violate the description in
	// TIA_HW_Notes.txt quoted above. the NUSIZ register can indeed be changed
	// at any time. if the sprite is currently been drawn the the tick()
	// function will immediately draw subsequent pixels immediately. the delay
	// only effects when the pixel drawing begins.

	ps.Delay.Schedule(delay, func() {
		// if size is 2x or 4x currently then take off the additional pixel. we'll
		// add it back on afterwards if needs be
		if ps.nusiz == 0x05 || ps.nusiz == 0x07 {
			ps.resetPixel--
			ps.hmovedPixel--
		}

		ps.nusiz = value & 0x07

		// if size is 2x or 4x then we need to record an additional pixel on the
		// reset point value
		if ps.nusiz == 0x05 || ps.nusiz == 0x07 {
			ps.resetPixel++
			ps.hmovedPixel++
		}

		// adjust reset pixel for screen boundaries
		if ps.resetPixel > ps.tv.GetSpec().ClocksPerVisible {
			ps.resetPixel -= ps.tv.GetSpec().ClocksPerVisible
		}
		if ps.hmovedPixel > ps.tv.GetSpec().ClocksPerVisible {
			ps.hmovedPixel -= ps.tv.GetSpec().ClocksPerVisible
		}
	}, "NUSIZx")
}

func (ps *playerSprite) setColor(value uint8) {
	// there is nothing in TIA_HW_Notes.txt about the color registers
	ps.color = value
}
