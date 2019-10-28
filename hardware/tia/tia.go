package tia

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/audio"
	"gopher2600/hardware/tia/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/hardware/tia/video"
	"gopher2600/television"
	"strings"
)

// TIA contains all the sub-components of the VCS TIA sub-system
type TIA struct {
	tv  television.Television
	mem memory.ChipBus

	// number of video cycles since the last WSYNC. also cycles back to 0 on
	// RSYNC and when polycounter reaches count 56
	//
	// cpu cycles can be attained by dividing videoCycles by 3
	videoCycles int

	// the last signal sent to the television. many signal attributes are
	// sustained over many cycles; we use this to store that information
	sig television.SignalAttributes

	// for clarity we think of tia video and audio as sub-systems
	Video *video.Video
	Audio *audio.Audio

	// horizontal blank controls whether to send colour information to the
	// television. it is turned on at the end of the visible screen and turned
	// on depending on the HMOVE latch. it is also used to control when sprite
	// counters are ticked.
	hblank bool

	// wsync records whether the cpu is to halt until hsync resets to 000000
	wsync bool

	// HMOVE information. each sprite object also contains HOMVE information
	// - hmoveLatch indicates whether HMOVE has been triggered this scanline.
	// it is reset when a new scanline begins
	hmoveLatch bool

	// - hmoveCt counts from 15 to 255. note that unlike how it is described in
	// TIA_HW_Notes.txt, we always send the extra tick to the sprites on Phi1.
	// however, we also send the hmoveCt value, whether or not the extra should
	// be honoured is up to the sprite. (TIA_HW_Notes.txt says that hmoveCt is
	// checked *before* sending the extra tick)
	hmoveCt uint8

	// TIA_HW_Notes.txt describes the hsync counter:
	//
	// "The HSync counter counts from 0 to 56 once for every TV scan-line
	// before wrapping around, a period of 57 counts at 1/4 CLK (57*4=228 CLK).
	// The counter decodes shown below provide all the horizontal timing for
	// the control lines used to construct a valid TV signal."
	hsync polycounter.Polycounter
	pclk  phaseclock.PhaseClock

	// TIA_HW_Notes.txt talks about there being a delay when altering some
	// video objects/attributes. the following future.Group ticks every color
	// clock. in addition to this, each sprite has it's own future.Group that
	// only ticks under certain conditions.
	Delay *future.Ticker

	// a reference to the delayed rsync event. we use this to determine if an
	// rsync has been scheduled and to hold off naturally occuring new
	// scanline events if it has
	rsyncEvent *future.Event

	// similarly for HMOVE events. we use this to help us decide whether we
	// have a late or early HBLANK
	hmoveEvent *future.Event
}

func (tia TIA) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s %s %d %04.01f",
		tia.hsync, tia.pclk,
		tia.videoCycles, float64(tia.videoCycles)/3.0,
	))

	if tia.hmoveCt != 0xff {
		s.WriteString(fmt.Sprintf(" hm=%04b", tia.hmoveCt))
	}

	return s.String()
}

// NewTIA creates a TIA, to be used in a VCS emulation
func NewTIA(tv television.Television, mem memory.ChipBus) *TIA {
	tia := TIA{tv: tv, mem: mem, hblank: true}

	tia.Delay = future.NewTicker("TIA")

	tia.pclk.Reset()
	tia.hmoveCt = 0xff

	tia.Video = video.NewVideo(&tia.pclk, &tia.hsync, mem, tv, &tia.hblank, &tia.hmoveLatch)
	if tia.Video == nil {
		return nil
	}

	tia.Audio = audio.NewAudio()
	if tia.Audio == nil {
		return nil
	}

	return &tia
}

// UpdateTIA checks for side effects in the TIA sub-system
func (tia *TIA) UpdateTIA(data memory.ChipData) {
	switch data.Name {
	case "VSYNC":
		tia.sig.VSync = data.Value&0x02 == 0x02

		// !!TODO: do something with controller settings below
		_ = data.Value&0x40 == 0x40
		_ = data.Value&0x80 == 0x80
		return

	case "VBLANK":
		// homebrew Donkey Kong shows the need for a delay of at least one
		// cycle for VBLANK. see area just before score box on play screen
		tia.Delay.Schedule(1, func() {
			tia.sig.VBlank = (data.Value&0x02 == 0x02)
		}, "VBLANK")
		return

	case "WSYNC":
		// CPU has indicated that it wants to wait for the beginning of the
		// next scanline. value is reset to false when TIA reaches end of
		// scanline
		tia.wsync = true
		return

	case "RSYNC":
		// from TIA_HW_Notes.txt:
		//
		// "RSYNC resets the two-phase clock for the HSync counter to the H@1
		// rising edge when strobed."
		tia.pclk.Align()

		// from TIA_HW_Notes.txt:
		//
		// "A full H@1-H@2 cycle after RSYNC is strobed, the HSync counter is
		// also reset to 000000 and HBlank is turned on."

		// the explanation as provided by TIA_HW_Notes was only of limited use.
		// the following delays were revealed by observation of Stella and how
		// it reacts to well known ROMs. In particular:
		//
		// * Pitfall - many ROMs clear the machine and hit RSYNC during
		// startup. I just happened to use Pitfall to see how the TV behaves
		// during startup
		//
		// * Extra Terrestrials - uses RSYNC to position ET correctly
		//
		// * Test RSYNC - test rom by Omegamatrix

		tia.rsyncEvent = tia.Delay.Schedule(3, func() {
			tia.newScanline()

			// adjust video elements by the number of visible pixels that have
			// been consumed. adding one to the value because the tv pixel we
			// want to hit has not been reached just yet
			adj, _ := tia.tv.GetState(television.ReqHorizPos)
			adj++
			if adj > 0 {
				tia.Video.RSYNC(adj)
			}

			tia.rsyncEvent = tia.Delay.Schedule(4, func() {
				tia.hsync.Reset()
				tia.pclk.Reset()
				tia.rsyncEvent = nil
			}, "RSYNC (reset)")
		}, "RSYNC (new scanline)")

		// I've not test what happens if we reach hsync naturally while the
		// above RSYNC delay is active.

		return

	case "HMOVE":
		// the scheduling for HMOVE is divided into two tranches, starting at
		// the same time:
		//
		// the TIA_HW_Notes.txt says this about HMOVE:
		//
		// "It takes 3 CLK after the HMOVE command is received to decode the
		// [SEC] signal (at most 6 CLK depending on the time of STA HMOVE) and
		// a further 4 CLK to set 'more movement required' latches."

		var delay int

		// not forgetting that we count from zero, the following delay
		// values range from 3 to 6, like the notes say
		switch tia.pclk.Count() {
		case 0:
			delay = 5
		case 1:
			delay = 4
		case 2:
			delay = 4
		case 3:
			delay = 2
		}

		tia.Delay.Schedule(delay, func() {
			tia.hmoveLatch = true
		}, "HMOVE")

		delay += 3

		tia.hmoveEvent = tia.Delay.Schedule(delay, func() {
			tia.Video.PrepareSpritesForHMOVE()
			tia.hmoveCt = 15
			tia.hmoveEvent = nil
		}, "HMOVE (mode movement latches)")

		// from TIA_HW_Notes:
		//
		// "Also of note, the HMOVE latch used to extend the HBlank time is
		// cleared when the HSync Counter wraps around. This fact is
		// exploited by the trick that invloves hitting HMOVE on the 74th
		// CPU cycle of the scanline; the CLK stuffing will still take
		// place during the HBlank and the HSYNC latch will be set just
		// before the counter wraps around. It will then be cleared again
		// immediately (and therefore ignored) when the counter wraps,
		// preventing the HMOVE comb effect."
		//
		// for the this "trick" to work correctly it's important that we get
		// the delay correct for pclk.Count() == 1 above. once that value had
		// been settled the other values fell into place.

		return
	}
}

func (tia *TIA) newScanline() {
	// the CPU's WSYNC concludes at the beginning of a scanline
	// from the TIA_1A document:
	//
	// "...WSYNC latch is automatically reset to zero by the
	// leading edge of the next horizontal blank timing signal,
	// releasing the RDY line"
	tia.wsync = false

	// start HBLANK. start of new scanline for the TIA. turn hblank
	// on
	tia.hblank = true

	// reset debugging information
	tia.videoCycles = 0

	// see SignalAttributes type definition for notes about the
	// HSyncSimple attribute
	tia.sig.HSyncSimple = true

	// rather than include the reset signal in the delay, we will
	// manually reset hsync counter when it reaches a count of 57
}

// Step moves the state of the tia forward one video cycle returns the state of
// the CPU (conceptually, we're attaching the result of this function to pin 3
// of the 6507)
//
// the meat of the Step() function can be divided into 8 sub-steps and 3 phases
// when the TIA state is altered in response to changes to TIA memory
//
// the ordering of these sub-steps is important. the currently defined steps
// and the ordering are as follows:
//
// A. service TIA memory / update playfield data
// 1. tick phase clock
// 2. tick delayed events
// 3. if phase clock is on the rising edge of Phi2
//		3.1. tick hsync counter
//		3.2. schedule hsync events as required
// B. service TIA video memory
// 4. tick video objects/events
// 5. adjust HMOVE value
// C. service TIA audio memory / late TIA video attributes
// 6. send signal to television
//
// step 5 contains a lot more work important to the correct operation of the
// TIA but from this perspective the step is monolithic
func (tia *TIA) Step(readMemory bool) (bool, error) {
	// update debugging information
	tia.videoCycles++

	var memoryData memory.ChipData

	// update memory if required
	if readMemory {
		readMemory, memoryData = tia.mem.ChipRead()
	}

	// make alterations to video state and playfield
	if readMemory {
		tia.UpdateTIA(memoryData)
		tia.Video.UpdatePlayfield(tia.Delay, memoryData)
	}

	// tick phase clock
	tia.pclk.Tick()

	// tick delayed events
	tia.Delay.Tick()

	// tick hsync counter when the Phi2 clock is raised. from TIA_HW_Notes.txt:
	//
	// "This table shows the elapsed number of CLK, CPU cycles, Playfield
	// (PF) bits and Playfield pixels at the start of each counter state
	// (ie when the counter changes to this state on the rising edge of
	// the H@2 clock)."
	//
	// the context of this passage is the Horizontal Sync Counter. It is
	// explicitely saying that the HSYNC counter ticks forward on the rising
	// edge of Phi2.
	if tia.pclk.Phi2() {
		tia.hsync.Tick()

		// hsyncDelay is the number of cycles required before, for example, hblank
		// is reset
		const hsyncDelay = 3

		// this switch statement is based on the "Horizontal Sync Counter"
		// table in TIA_HW_Notes.txt. the "key" at the end of that table
		// suggests that (most of) the events are delayed by 4 clocks due to
		// "latching".
		switch tia.hsync.Count {
		case 57:
			// from TIA_HW_Notes.txt:
			//
			// "The HSync counter resets itself after 57 counts; the decode on
			// HCount=56 performs a reset to 000000 delayed by 4 CLK, so
			// HCount=57 becomes HCount=0. This gives a period of 57 counts
			// or 228 CLK."
			tia.hsync.Reset()

			// from TIA_HW_Notes.txt:
			//
			// "Also of note, the HMOVE latch used to extend the HBlank time
			// is cleared when the HSync Counter wraps around. This fact is
			// exploited by the trick that invloves hitting HMOVE on the 74th
			// CPU cycle of the scanline; the CLK stuffing will still take
			// place during the HBlank and the HSYNC latch will be set just
			// before the counter wraps around."
			tia.hmoveLatch = false

		case 56: // [SHB]
			// allow a new scanline event to occur naturally only when an RSYNC
			// has not been scheduled
			if tia.rsyncEvent == nil {
				tia.Delay.Schedule(hsyncDelay, func() {
					tia.newScanline()
				}, "RESET")
			}

		case 4: // [SHS]
			// start HSYNC. start of new scanline for the television
			// * TIA_HW_Notes.txt does not say there is a 4 clock delay for
			// this. not clear if this is the case.
			//
			// !!TODO: check accuracy of HSync timing
			tia.sig.HSync = true

		case 8: // [RHS]
			tia.Delay.Schedule(hsyncDelay, func() {
				// reset HSYNC
				tia.sig.HSync = false
				tia.sig.CBurst = true
			}, "RHS (TV)")

		case 12: // [RCB]
			tia.Delay.Schedule(hsyncDelay, func() {
				// reset color burst
				tia.sig.CBurst = false
			}, "RCB (TV)")

		// the two cases below handle the turning off of the hblank flag. from
		// TIA_HW_Notes.txt:
		//
		// "In principle the operation of HMOVE is quite straight-forward; if a
		// HMOVE is initiated immediately after HBlank starts, which is the
		// case when HMOVE is used as documented, the [HMOVE] signal is latched
		// and used to delay the end of the HBlank by exactly 8 CLK, or two
		// counts of the HSync Counter. This is achieved in the TIA by
		// resetting the HB (HBlank) latch on the [LRHB] (Late Reset H-Blank)
		// counter decode rather than the normal [RHB] (Reset H-Blank) decode."

		// in practice we have to careful about when HMOVE has been triggered.
		// the condition below for HSYNC=16 includes a test for an active HMOVE
		// event and whether it is about to be completed. we can see the effect
		// of this in particular in the test ROM "games that do bad thing to
		// HMOVE" at value 14

		case 16: // [RHB]
			// early HBLANK off if hmoveLatch is false
			if !tia.hmoveLatch {
				tia.Delay.Schedule(hsyncDelay, func() {
					tia.hblank = false
				}, "HRB")
			}

		// ... and "two counts of the HSync Counter" later ...

		case 18:
			// late HBLANK off if hmoveLatch is true
			if tia.hmoveLatch {
				tia.Delay.Schedule(hsyncDelay, func() {
					tia.hblank = false
				}, "LHRB")
			}
		}
	}

	// alter state of video subsystem. occuring after ticking of TIA clock
	// because some the side effects of some registers require that. in
	// particular, the RESxx registers need to have correct information about
	// the state of HBLANK and the HMOVE latch.
	//
	// to see the effect of this, try moving this function call before the
	// HSYNC tick and see how the ball sprite is rendered incorrectly in
	// Keystone Kapers (this is because the ball is reset on the very last
	// pixel and before HBLANK etc. are in the state they need to be)
	if readMemory {
		tia.Video.UpdateSpritePositioning(memoryData)
		tia.Video.UpdateColor(memoryData)
	}

	// "one extra CLK pulse is sent every 4 CLK" and "on every H@1 signal [...]
	// as an extra 'stuffed' clock signal."
	isHmove := tia.pclk.Phi2()

	// we always call TickSprites but whether or not (and how) the tick
	// actually occurs is left for the sprite object to decide based on the
	// arguments passed here.
	tia.Video.Tick(!tia.hblank, isHmove, tia.hmoveCt)

	// update hmove counter value
	if isHmove {
		if tia.hmoveCt != 0xff {
			tia.hmoveCt--
		}
	}

	// resolve video pixels. note that we always send the debug color
	// regardless of hblank
	pixelColor, debugColor := tia.Video.Pixel()
	tia.sig.AltPixel = television.ColorSignal(debugColor)
	if tia.hblank {
		// if hblank is on then we don't sent the resolved color but the video
		// black signal instead
		tia.sig.Pixel = television.VideoBlack
	} else {
		tia.sig.Pixel = television.ColorSignal(pixelColor)
	}

	if readMemory {
		tia.Video.UpdateSpriteHMOVE(tia.Delay, memoryData)
		tia.Video.UpdateSpriteVariations(memoryData)
		tia.Video.UpdateSpritePixels(memoryData)

		// update audio signal
		tia.Audio.UpdateOutput(memoryData)
	}

	// copy audio to television signal
	tia.sig.Audio = *tia.Audio

	// send signal to television
	if err := tia.tv.Signal(tia.sig); err != nil {
		// allow out-of-spec errors for now. this should be optional
		if !errors.Is(err, errors.TVOutOfSpec) {
			return !tia.wsync, err
		}
	}

	// reset the HSyncSimple attribute as soon as the signal has been sent
	// (see SignalAttributes type definition for notes about the HSyncSimple
	// attribute)
	tia.sig.HSyncSimple = false

	return !tia.wsync, nil
}
