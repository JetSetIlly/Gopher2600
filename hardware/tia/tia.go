package tia

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/audio"
	"gopher2600/hardware/tia/delay/future"
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

	// number of cycles since the last WSYNC
	cpuCycles   float64
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
	hmoveLatch bool
	hmoveCt    int

	// "Beside each counter there is a two-phase clock generator. This
	// takes the incoming 3.58 MHz colour clock (CLK) and divides by
	// 4 using a couple of flip-flops. Two AND gates are then used to
	// generate two independent clock signals"
	//
	// we use tiaClk by waiting for InPhase() signals and then ticking the
	// hsync counter.
	tiaClk phaseclock.PhaseClock

	// TIA_HW_Notes.txt describes the hsync counter:
	//
	// "The HSync counter counts from 0 to 56 once for every TV scan-line
	// before wrapping around, a period of 57 counts at 1/4 CLK (57*4=228 CLK).
	// The counter decodes shown below provide all the horizontal timing for
	// the control lines used to construct a valid TV signal."
	hsync polycounter.Polycounter

	// TIA_HW_Notes.txt talks about there being a delay when altering some
	// video objects/attributes. the following future.Group ticks every color
	// clock. in addition to this, each sprite has it's own future.Group that
	// only ticks under certain conditions.
	TIAdelay future.Ticker
}

// MachineInfoTerse returns the TIA information in terse format
func (tia TIA) MachineInfoTerse() string {
	return tia.String()
}

// MachineInfo returns the TIA information in verbose format
func (tia TIA) MachineInfo() string {
	return tia.String()
}

// map String to MachineInfo
func (tia TIA) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s %s %03d %04.01f %d",
		tia.hsync,
		tia.tiaClk.String(),
		tia.videoCycles,
		tia.cpuCycles,

		// pixel information below is not the same as the pixel column in
		// TIA_HW_Notes
		tia.hsync.NumSteps(&tia.tiaClk),
	))

	// NOTE: TIA_HW_Notes also includes playfield and control information.
	// we're choosing not to include that information here

	return s.String()
}

// NewTIA creates a TIA, to be used in a VCS emulation
func NewTIA(tv television.Television, mem memory.ChipBus) *TIA {
	tia := TIA{tv: tv, mem: mem, hblank: true}

	tia.tiaClk.Reset(false)
	tia.hmoveCt = -1

	tia.Video = video.NewVideo(&tia.tiaClk, &tia.hsync, &tia.TIAdelay, mem, tv)
	if tia.Video == nil {
		return nil
	}

	tia.Audio = audio.NewAudio()
	if tia.Audio == nil {
		return nil
	}

	return &tia
}

// ReadMemory checks for side effects in the TIA sub-system
func (tia *TIA) ReadMemory() {
	service, register, value := tia.mem.ChipRead()

	if !service {
		// nothing to service
		return
	}

	switch register {
	case "VSYNC":
		tia.sig.VSync = value&0x02 == 0x02

		// !!TODO: do something with controller settings below
		_ = value&0x40 == 0x40
		_ = value&0x80 == 0x80
		return

	case "VBLANK":
		tia.sig.VBlank = (value&0x02 == 0x02)
		return

	case "WSYNC":
		// CPU has indicated that it wants to wait for the beginning of the
		// next scanline. value is reset to false when TIA reaches end of
		// scanline
		tia.wsync = true
		return

	case "RSYNC":
		tia.tiaClk.Reset(true)
		tia.TIAdelay.Schedule(5, func() {
			tia.hsync.Reset()

			// the same as what happens at SHB
			tia.hblank = true
			tia.wsync = false
			tia.hmoveLatch = false
			tia.videoCycles = 0
			tia.cpuCycles = 0
			tia.sig.HSyncSimple = true
		}, "RSYNC")
		return

	case "HMOVE":
		tia.Video.PrepareSpritesForHMOVE()
		tia.hmoveLatch = true
		tia.hmoveCt = 15
		return
	}

	if tia.Video.ReadMemory(register, value) {
		return
	}

	if tia.Audio.ReadMemory(register, value) {
		return
	}

	panic(fmt.Sprintf("unserviced register (%s=%v)", register, value))
}

// Step moves the state of the tia forward one video cycle returns the state of
// the CPU (conceptually, we're attaching the result of this function to pin 3
// of the 6507)
//
// the meat of the Step() function can be divided into 9 parts. the ordering of
// these parts is important. the currently defined steps and the ordering are
// as follows:
//
// !!TODO: summary of steps
//
// steps 2.0 and 6.0 contain a lot more work important to the correct operation
// of the TIA but from this perspective each step is monolithic
//
// note that there is no TickPlayfield(). earlier versions of the code required
// us to tick the playfield explicitely but because the playfield is so closely
// tied to the hysnc counter it was decided to make the ticking implicit.
// removing the redundent moving part made the ordering of the individual steps
// obvious.
func (tia *TIA) Step() (bool, error) {
	// update debugging information
	tia.videoCycles++
	tia.cpuCycles = float64(tia.videoCycles) / 3.0

	// update "two-phase clock generator"
	tia.tiaClk.Tick()

	// hsyncDelay is the number of cycles required before, for example, hblank
	// is reset
	const hsyncDelay = 4

	// when phase clock reaches the correct state, tick hsync counter
	if tia.tiaClk.InPhase() {
		tia.hsync.Tick()

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

		case 56: // [SHB]
			tia.TIAdelay.Schedule(hsyncDelay, func() {
				// the CPU's WSYNC concludes at the beginning of a scanline
				// from the TIA_1A document:
				//
				// "...WYNC latch is automatically reset to zero by the leading
				// edge of the next horizontal blank timing signal, releasing
				// the RDY line"
				//
				// the reutrn value of this Step() function is the RDY line
				tia.wsync = false

				// start HBLANK. start of new scanline for the TIA. turn hblank on
				tia.hblank = true

				// not sure when to reset HMOVE latch but here seems good
				tia.hmoveLatch = false

				// reset debugging information
				tia.videoCycles = 0
				tia.cpuCycles = 0

				// rather than include the reset signal in the delay, we will
				// manually reset hsync counter when it reaches a count of 57

				// see SignalAttributes type definition for notes about the
				// HSyncSimple attribute
				tia.sig.HSyncSimple = true
			}, "RESET")

		case 1:
			// reset the HSyncSimple attribute as soon as is practical
			//
			// see SignalAttributes type definition for notes about the
			// HSyncSimple attribute
			tia.sig.HSyncSimple = false

		case 4: // [SHS]
			// start HSYNC. start of new scanline for the television
			// * TIA_HW_Notes.txt does not say there is a 4 clock delay for
			// this even
			tia.sig.HSync = true

		case 8: // [RHS]
			tia.TIAdelay.Schedule(hsyncDelay, func() {
				// reset HSYNC
				tia.sig.HSync = false
				tia.sig.CBurst = true
			}, "RHS (TV)")

		case 12: // [RCB]
			tia.TIAdelay.Schedule(hsyncDelay, func() {
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

		case 16: // [RHB]
			// early HBLANK off if hmoveLatch is false
			if !tia.hmoveLatch {
				tia.TIAdelay.Schedule(hsyncDelay, func() {
					tia.hblank = false
				}, "HRB")
			}

		// ... and "two counts of the HSync Counter" later ...

		case 18:
			// late HBLANK off if hmoveLatch is true
			if tia.hmoveLatch {
				tia.TIAdelay.Schedule(hsyncDelay, func() {
					tia.hblank = false
				}, "LHRB")
			}
		}
	}

	// tick future events. ticking here so that it happens at the same time as
	// the regular TIA clock. this is important because then the delay values
	// work out as you would expect.
	//
	// (this may seem like an obvious fact but I spent enough time fiddling with
	// this and convincing myself that it was true that it seems worthy of a
	// note)
	tia.TIAdelay.Tick()

	// we always call TickSprites but whether or not (and how) the tick
	// actually occurs is left for the sprite object to decide based on the
	// arguments passed here.
	//
	// the first argument is whether or not we're in the visible part of the
	// screen. from TIA_HW_Notes.txt:
	//
	// "The most important thing to note about the player counter is
	// that it only receives CLK signals during the visible part of
	// each scanline, when HBlank is off; exactly 160 CLK per scanline
	// (except during HMOVE)"
	//
	// from this we can say that the concept of the visible screen coincides
	// exactly with when HBLANK is disabled.
	//
	// the second argument is the current hmove counter value. from
	// TIA_HW_Notes.txt:
	//
	// "In this case the extra HMOVE clock pulses act to perform
	// 'plugging' instead of the normal 'stuffing'; by this I mean that
	// the extra pulses plug up the gaps in the normal [MOTCK] pulses,
	// preventing them from counting as clock pulses. This only works
	// because the extra HMOVE pulses are derived from the two-phase
	// clock on the HSync counter, which is itself derived from CLK
	// (the TIA colour clock input), whereas [MOTCK] is an inverted CLK
	// signal - so they are more or less precisely out of phase :)"
	tia.Video.TickSprites(!tia.hblank, uint8(tia.hmoveCt)&0x0f)

	// update HMOVE counter. leaving the value as -1 (the binary for -1 is of
	// course 0b11111111)
	if tia.hmoveCt >= 0 {
		tia.hmoveCt--
	}

	// resolve video pixels. note that we always send the debug color
	// regardless of hblank
	pixelColor, debugColor := tia.Video.Resolve()
	tia.sig.AltPixel = television.ColorSignal(debugColor)
	if tia.hblank {
		// if hblank is on then we don't sent the resolved color but the video
		// black signal instead
		tia.sig.Pixel = television.VideoBlack
	} else {
		tia.sig.Pixel = television.ColorSignal(pixelColor)
	}

	// send signal to television
	if err := tia.tv.Signal(tia.sig); err != nil {
		switch err := err.(type) {
		case errors.FormattedError:
			// filter out-of-spec errors for now. this should be optional
			if err.Errno != errors.TVOutOfSpec {
				return !tia.wsync, err
			}
		default:
			return !tia.wsync, err
		}
	}

	return !tia.wsync, nil
}
