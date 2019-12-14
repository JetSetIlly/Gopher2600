package video

import (
	"fmt"
	"gopher2600/hardware/tia/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/television"
	"strings"
)

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
	position *polycounter.Polycounter

	// "Beside each counter there is a two-phase clock generator..."
	pclk phaseclock.PhaseClock

	// each sprite keeps track of its own delays. this way, we can carefully
	// control when the sprite events occur - taking into consideration sprite
	// specific conditions
	//
	// note that setGfxData() uses the TIA wide future instance
	Delay *future.Ticker

	// horizontal movement
	moreHMOVE bool
	hmove     uint8

	// the last hmovect value seen by the Tick() function. used to accurately
	// decide the delay period when resetting the sprite position
	lastHmoveCt uint8

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

	// ^^^ the above are common to all sprite types ^^^

	// player sprite attributes
	color         uint8
	nusiz         uint8
	reflected     bool
	verticalDelay bool
	gfxDataNew    uint8
	gfxDataOld    uint8

	// pointer to which gfx data we're using (gfxDataOld or gfxDataNew).
	// controlled by value of verticalDelay
	gfxData *uint8

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

	// reference to ball sprite. only required by player1 sprite. see
	// setGfxData() function below
	ball *ballSprite

	// a record of the delayed start drawing event. resets to nil once drawing
	// commences
	startDrawingEvent *future.Event

	// a record of the delayed reset event. resets to nil once reset has
	// occurred
	resetPositionEvent *future.Event
}

func newPlayerSprite(label string, tv television.Television, hblank, hmoveLatch *bool) *playerSprite {
	ps := playerSprite{
		label:      label,
		tv:         tv,
		hblank:     hblank,
		hmoveLatch: hmoveLatch,
	}

	var err error

	ps.position, err = polycounter.New(6)
	if err != nil {
		// TODO: propogate this error upwards
		return nil
	}

	ps.Delay = future.NewTicker(label)

	ps.scanCounter.nusiz = &ps.nusiz
	ps.scanCounter.pclk = &ps.pclk
	ps.position.Reset()

	// initialise gfxData pointer
	ps.gfxData = &ps.gfxDataNew

	return &ps
}

func (ps playerSprite) String() string {
	// the hmove value as maintained by the sprite type is normalised for
	// for purposes of presentation
	normalisedHmove := int(ps.hmove) - 8
	if normalisedHmove < 0 {
		normalisedHmove = 16 + normalisedHmove
	}

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s: ", ps.label))
	s.WriteString(fmt.Sprintf("%s %s [%03d ", ps.position, ps.pclk, ps.resetPixel))
	s.WriteString(fmt.Sprintf("> %#1x >", normalisedHmove))
	s.WriteString(fmt.Sprintf(" %03d", ps.hmovedPixel))
	if ps.moreHMOVE {
		s.WriteString("*] ")
	} else {
		s.WriteString("] ")
	}

	// interpret nusiz value - using scancounter's latchedNusiz value
	switch ps.scanCounter.latchedNusiz {
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

	// add a note to indicate that the nusiz value is about to update
	if ps.nusiz != ps.scanCounter.latchedNusiz {
		s.WriteString(" *")
	}

	// notes
	extra := false

	if ps.moreHMOVE {
		s.WriteString(" hmoving")
		s.WriteString(fmt.Sprintf(" [%04b]", ps.hmove))
		extra = true
	}

	if ps.scanCounter.isActive() {
		// add a comma if we've already noted something else
		if extra {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw (px %d", ps.scanCounter.pixel))
		if ps.scanCounter.count > 0 {
			// add "sub-pixel" information
			s.WriteString(fmt.Sprintf(".%d", ps.scanCounter.count))
		}
		s.WriteString(")")
		extra = true

		switch ps.scanCounter.cpy {
		case 1:
			s.WriteString(" near")
		case 2:
			s.WriteString(" far")
		}

	} else if ps.scanCounter.isLatching() {
		// add a comma if we've already noted something else
		if extra {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw (in %d)", ps.scanCounter.latch))
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
	}

	// 	s.WriteString("\n")
	// 	s.WriteString(fmt.Sprintf("gfx new: %08b", ps.gfxDataNew))
	// 	if !ps.verticalDelay {
	// 		s.WriteString(" *")
	// 	}
	// 	s.WriteString("\n")
	// 	s.WriteString(fmt.Sprintf("gfx old: %08b", ps.gfxDataOld))
	// 	if ps.verticalDelay {
	// 		s.WriteString(" *")
	// 	}

	return s.String()
}

func (ps *playerSprite) rsync(adjustment int) {
	ps.resetPixel -= adjustment
	ps.hmovedPixel -= adjustment
	if ps.resetPixel < 0 {
		ps.resetPixel += television.HorizClksVisible
	}
	if ps.hmovedPixel < 0 {
		ps.hmovedPixel += television.HorizClksVisible
	}
}

// tick moves the sprite counters along (both position and graphics scan).
func (ps *playerSprite) tick(visible, isHmove bool, hmoveCt uint8) {
	// check to see if there is more movement required for this sprite
	if isHmove {
		ps.moreHMOVE = ps.moreHMOVE && compareHMOVE(hmoveCt, ps.hmove)
	}

	ps.lastHmoveCt = hmoveCt

	// early return if nothing to do
	if !(isHmove && ps.moreHMOVE) && !visible {
		return
	}

	// update hmoved pixel value
	if !visible {
		ps.hmovedPixel--

		// adjust for screen boundary
		if ps.hmovedPixel < 0 {
			ps.hmovedPixel += television.HorizClksVisible
		}
	}

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
	// the Graphics Scan Counter. Depending on the player stretch mode, one
	// clock signal is allowed through every 1, 2 or 4 graphics CLK.  The
	// stretched modes are derived from the two-phase clock; the H@2 phase
	// allows 1 in 4 CLK through (4x stretch), both phases ORed together
	// allow 1 in 2 CLK through (2x stretch)."
	ps.scanCounter.tick()

	// tick phase clock after scancounter tick
	ps.pclk.Tick()

	// I cannot find a direct reference that describes when position
	// counters are ticked forward. however, TIA_HW_Notes.txt does say the
	// HSYNC counter ticks forward on the rising edge of Phi2. it is
	// reasonable to assume that the sprite position counters do likewise.
	if ps.pclk.Phi2() {
		ps.position.Tick()

		// drawing must not start if a reset position event has been
		// recently scheduled.
		//
		// rules discovered through observation (games that do bad things
		// to HMOVE)
		if ps.resetPositionEvent == nil || ps.resetPositionEvent.JustStarted() {
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

			// "... The START decodes are ANDed with flags from the NUSIZ register
			// before being latched, to determine whether to draw that copy."
			switch ps.position.Count() {
			case 3:
				if ps.nusiz == 0x01 || ps.nusiz == 0x03 {
					ps.startDrawingEvent = ps.Delay.ScheduleWithArg(4, ps._futureStartDrawingEvent, 1, "START")
				}
			case 7:
				if ps.nusiz == 0x03 || ps.nusiz == 0x02 || ps.nusiz == 0x06 {
					cpy := 1
					if ps.nusiz == 0x03 {
						cpy = 2
					}
					ps.startDrawingEvent = ps.Delay.ScheduleWithArg(4, ps._futureStartDrawingEvent, cpy, "START")
				}
			case 15:
				if ps.nusiz == 0x04 || ps.nusiz == 0x06 {
					cpy := 1
					if ps.nusiz == 0x06 {
						cpy = 2
					}
					ps.startDrawingEvent = ps.Delay.ScheduleWithArg(4, ps._futureStartDrawingEvent, cpy, "START")
				}
			case 39:
				ps.startDrawingEvent = ps.Delay.ScheduleWithArg(4, ps._futureStartDrawingEvent, 0, "START")

			case 40:
				ps.position.Reset()
			}
		}
	}

	// tick future events that are goverened by the sprite
	ps.Delay.Tick()
}

func (ps *playerSprite) _futureStartDrawingEvent(v interface{}) {
	// it is useful for debugging to know which copy of the sprite is
	// currently being drawn. we'll update this value in the switch
	// below, taking great care to note the value of ms.copies at each
	// trigger point
	//
	// this is used by the missile sprites when in reset-to-player
	// mode
	ps.scanCounter.cpy = v.(int)

	ps.scanCounter.start()
	ps.startDrawingEvent = nil
}

func (ps *playerSprite) prepareForHMOVE() {
	// the latching delay should already have been consumed when servicing the
	// HMOVE signal in tia.go

	ps.moreHMOVE = true

	if *ps.hblank {
		// adjust hmovedPixel value. this value is subject to further change so
		// long as moreHMOVE is true. the String() function this value is
		// annotated with a "*" to indicate that HMOVE is still in progress
		ps.hmovedPixel += 8

		// adjust for screen boundary
		if ps.hmovedPixel > television.HorizClksVisible {
			ps.hmovedPixel -= television.HorizClksVisible
		}
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
		// this tricky branch happens when reset is triggered inside the
		// HBLANK period and HMOVE is active. in this instance we're defining
		// active to be whether the last HmoveCt value was between 15 and 0
		if !*ps.hmoveLatch || ps.lastHmoveCt >= 1 && ps.lastHmoveCt <= 15 {
			delay = 2
		} else {
			delay = 3
		}
	}

	// pause pending start drawing events unless it is about to start this
	// cycle
	//
	// rules discovered through observation (games that do bad things
	// to HMOVE)
	if ps.startDrawingEvent != nil && !ps.startDrawingEvent.AboutToEnd() {
		ps.startDrawingEvent.Pause()
	}

	// stop any existing reset events. generally, this codepath will not apply
	// because a resetPositionEvent will conculde before being triggered again.
	// but it is possible when using a very quick instruction on the reset register,
	// like a zero page INC, for requests to overlap
	if ps.resetPositionEvent != nil {
		ps.resetPositionEvent.Push()
		return
	}

	ps.resetPositionEvent = ps.Delay.Schedule(delay, ps._futureResetPosition, "RESPx")
}

func (ps *playerSprite) _futureResetPosition() {
	// the pixel at which the sprite has been reset, in relation to the
	// left edge of the screen
	ps.resetPixel, _ = ps.tv.GetState(television.ReqHorizPos)

	if ps.resetPixel >= 0 {
		// resetPixel adjusted by +1 because the tv is not yet in the correct.
		// position. and another +1 because of the latching required before
		// player sprites begin drawing
		ps.resetPixel += 2

		// if size is 2x or 4x then we need an additional reset pixel
		//
		// note that we need to monkey with resetPixel whenever NUSIZ changes.
		// see setNUSIZ() function below
		if ps.nusiz == 0x05 || ps.nusiz == 0x07 {
			ps.resetPixel++
		}

		// adjust resetPixel for screen boundaries
		if ps.resetPixel > television.HorizClksVisible {
			ps.resetPixel -= television.HorizClksVisible
		}

		// by definition the current pixel is the same as the reset pixel at
		// the moment of reset
		ps.hmovedPixel = ps.resetPixel
	} else {
		// if reset occurs off-screen then force reset pixel to be zero
		// (see commentary in ball sprite for detailed reasoning of this
		// branch)
		ps.resetPixel = 0
		ps.hmovedPixel = 7
	}

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
	// rules discovered through observation (games that do bad things
	// to HMOVE)
	if ps.startDrawingEvent != nil {
		if !ps.startDrawingEvent.JustStarted() {
			ps.startDrawingEvent.Force()
			ps.startDrawingEvent = nil
		} else {
			ps.startDrawingEvent.Drop()
			ps.startDrawingEvent = nil
		}
	}

	// dump reference to reset event
	ps.resetPositionEvent = nil
}

// pixel returns the color of the player at the current time.  returns
// (false, col) if no pixel is to be seen; and (true, col) if there is
func (ps *playerSprite) pixel() (bool, uint8) {
	// pick the pixel from the gfxData register
	if ps.scanCounter.isActive() {
		var offset int

		if ps.reflected {
			offset = 7 - ps.scanCounter.pixel
		} else {
			offset = ps.scanCounter.pixel
		}

		if *ps.gfxData>>offset&0x01 == 0x01 {
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
	ps.otherPlayer.gfxDataOld = ps.otherPlayer.gfxDataNew
	ps.gfxDataNew = data

	// if player sprite is connected to the ball sprite then update the delayed
	// output for the ball. only used by player1 sprite.
	if ps.ball != nil {
		ps.ball.setEnableDelay()
	}
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

	if ps.verticalDelay {
		ps.gfxData = &ps.gfxDataOld
	} else {
		ps.gfxData = &ps.gfxDataNew
	}
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

// !!TODO: the setNUSIZ() function needs untangling. I reckon with a bit of
// reordering we can simplify it quite a bit

func (ps *playerSprite) setNUSIZ(value uint8) {
	// from TIA_HW_Notes.txt:
	//
	// "The NUSIZ register can be changed at any time in order to alter
	// the counting frequency, since it is read every graphics CLK.
	// This should allow possible player graphics warp effects etc."

	// whilst the notes say that the register can be changed at any time, there
	// is a delay of sorts in certain situations; although  under most
	// circumstances, TIA_HW_Notes is correct, there is no delay.
	//
	// for convenience, we still call the Schedule() function but with a delay
	// value of -1 (see Schedule() function notes)
	delay := -1

	if ps.startDrawingEvent != nil {
		if ps.nusiz == 0x05 || ps.nusiz == 0x07 {
			delay = 0
		} else if ps.startDrawingEvent.RemainingCycles == 0 {
			delay = 1
		} else if ps.startDrawingEvent.RemainingCycles >= 2 &&
			ps.nusiz != value && ps.nusiz != 0x00 &&
			(value == 0x05 || value == 0x07) {

			// this branch applies when a sprite is changing from a single
			// width sprite to a double/quadruple width sprite. in that
			// instance we drop the drawing event if it has only recently
			// started
			//
			// I'm not convinced by this branch at all but the rule was
			// discovered through observation and balancing of the test roms:
			//
			//  o player_switching.bin
			//	o testSize2Copies_A.bin
			//	o properly_model_nusiz_during_player_decode_and_draw/player8.bin
			//
			// the rules maybe more subtle or more general than this
			ps.startDrawingEvent.Drop()
			ps.startDrawingEvent = nil
		}
	} else if ps.scanCounter.isLatching() || ps.scanCounter.isActive() {
		if (ps.nusiz == 0x05 || ps.nusiz == 0x07) && (value == 0x05 || value == 0x07) {
			// minimal delay current if future/current NUSIZ is double/quadruple width
			delay = 0
		} else {
			delay = 1
		}
	}

	ps.Delay.ScheduleWithArg(delay, ps._futureSetNusiz, value, "NUSIZx")
}

func (ps *playerSprite) _futureSetNusiz(v interface{}) {
	// if size is 2x or 4x currently then take off the additional pixel. we'll
	// add it back on afterwards if needs be
	if ps.nusiz == 0x05 || ps.nusiz == 0x07 {
		ps.resetPixel--
		ps.hmovedPixel--
	}

	ps.nusiz = v.(uint8) & 0x07

	// if size is 2x or 4x then we need to record an additional pixel on the
	// reset point value
	if ps.nusiz == 0x05 || ps.nusiz == 0x07 {
		ps.resetPixel++
		ps.hmovedPixel++
	}

	// adjust reset pixel for screen boundaries
	if ps.resetPixel > television.HorizClksVisible {
		ps.resetPixel -= television.HorizClksVisible
	}
	if ps.hmovedPixel > television.HorizClksVisible {
		ps.hmovedPixel -= television.HorizClksVisible
	}
}

func (ps *playerSprite) setColor(value uint8) {
	// there is nothing in TIA_HW_Notes.txt about the color registers
	ps.color = value
}

func (ps *playerSprite) setHmoveValue(v interface{}) {
	ps.hmove = (v.(uint8) ^ 0x80) >> 4
}

func (ps *playerSprite) clearHmoveValue() {
	ps.hmove = 0x08
}
