// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package tia

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/hardware/tia/delay"
	"github.com/jetsetilly/gopher2600/hardware/tia/hmove"
	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
	"github.com/jetsetilly/gopher2600/hardware/tia/polycounter"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
	"github.com/jetsetilly/gopher2600/logger"
)

// CPU defines the CPU functions required by the TIA type.
type CPU interface {
	SetRDY(bool)
}

// TV defines the television functions required by the TIA type.
type TV interface {
	Signal(signal.SignalAttributes)
	AudioSignal(signal.AudioSignalAttributes)
	GetCoords() coords.TelevisionCoords
}

// RIOTports is used to connect the TIA with the RIOT Ports implementation. We
// need this because the RIOT is affected by one of the bits in the VBLANK
// signal but because VBLANK is in the TIA address space we need to forward it
// from here.
type RIOTports interface {
	Update(chipbus.ChangedRegister) bool
}

// TIA contains all the sub-components of the VCS TIA sub-system.
type TIA struct {
	env *environment.Environment

	cpu CPU
	tv  TV

	// the VBLANK register also affects the RIOT sub-system
	riot RIOTports

	// number of color clocks since the last CPU cycle boundary
	ClocksSinceCycle int

	// the last signal sent to the television. many signal attributes are
	// sustained over many cycles; we use this to store that information
	sig signal.SignalAttributes

	// for clarity we think of tia video and audio as sub-systems
	Video *video.Video
	Audio *audio.Audio

	// horizontal blank controls whether to send colour information to the
	// television. it is turned on at the end of the visible screen and turned
	// on depending on the HMOVE latch. it is also used to control when sprite
	// counters are ticked.
	Hblank bool

	// Hmove information
	Hmove hmove.Hmove

	// TIA_HW_Notes.txt describes the hsync counter:
	//
	// "The HSync counter counts from 0 to 56 once for every TV scan-line
	// before wrapping around, a period of 57 counts at 1/4 CLK (57*4=228 CLK).
	// The counter decodes shown below provide all the horizontal timing for
	// the control lines used to construct a valid TV signal."
	hsync polycounter.Polycounter
	PClk  phaseclock.PhaseClock

	// some events are delayed. note that there are delay.Event instances in
	// the Hmove type.
	futureVblank     delay.Event
	futureRsyncAlign delay.Event
	futureRsyncReset delay.Event

	// hsync is a bit different because the semantics can change. hysnc events
	// never overlap so one delay.Event instance is sufficient. the
	// futureHsyncEvent field is used to differentiate.
	futureHsync      delay.Event
	futureHsyncEvent hsyncEvent

	// whether there are any delay.Events outstanding. if pendingEvents is zero
	// then we don't need to call resolveDelayedEvents()
	pendingEvents int
}

// a hsyncEvent is one that is triggered by the hsync polycounter
type hsyncEvent int

// list of valid hsyncEvent values
const (
	hsyncEventSHB hsyncEvent = iota
	hsyncEventRHS
	hsyncEventRCB
	hsyncEventRHB
	hsyncEventLRHB
)

// Label returns an identifying label for the TIA.
func (tia *TIA) Label() string {
	return "TIA"
}

func (tia *TIA) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s %s",
		tia.hsync, tia.PClk,
	))
	return s.String()
}

// NewTIA creates a TIA, to be used in a VCS emulation.
func NewTIA(env *environment.Environment, tv TV, mem chipbus.Memory, riot RIOTports, cpu CPU) (*TIA, error) {
	tia := &TIA{
		env:    env,
		cpu:    cpu,
		tv:     tv,
		riot:   riot,
		Hblank: true,
	}

	tia.Audio = audio.NewAudio(env)
	tia.Video = video.NewVideo(env, mem, tv, &tia.PClk, &tia.hsync, &tia.Hblank, &tia.Hmove)
	tia.Hmove.Reset()
	tia.PClk = phaseclock.ResetValue

	return tia, nil
}

// there is not reset function for the TIA because (a) it would be just too
// large an effort and (b) creating a new TIA instance is just as effective.

// Snapshot creates a copy of the TIA in its current state.
func (tia *TIA) Snapshot() *TIA {
	n := *tia
	n.Audio = tia.Audio.Snapshot()
	n.Video = tia.Video.Snapshot()
	return &n
}

// Plumb the a new ChipBus into the TIA.
func (tia *TIA) Plumb(env *environment.Environment, tv TV, mem chipbus.Memory, riot RIOTports, cpu CPU) {
	tia.env = env
	tia.cpu = cpu
	tia.tv = tv
	tia.riot = riot
	tia.Video.Plumb(tia.env, mem, tia.tv, &tia.PClk, &tia.hsync, &tia.Hblank, &tia.Hmove)
	tia.Audio.Plumb(tia.env)
}

// Update checks to see if ChipData applies tot he TIA and updates accordingly.
//
// Returns true if ChipData has *not* been serviced.
func (tia *TIA) Update(reg chipbus.ChangedRegister) bool {
	switch reg.Register {
	case cpubus.VSYNC:
		tia.sig.VSync = reg.Value&0x02 == 0x02
		return false

	case cpubus.VBLANK:
		// homebrew Donkey Kong shows the need for a delay of at least one
		// cycle for VBLANK. see area just before score box on play screen
		tia.futureVblank.Schedule(1, reg.Value)
		tia.pendingEvents++

		// the VBLANK register also affects the RIOT riot sub-system
		tia.riot.Update(reg)

		return false

	case cpubus.WSYNC:
		// CPU has indicated that it wants to wait for the beginning of the
		// next scanline. value is reset to false when TIA reaches end of
		// scanline
		tia.cpu.SetRDY(false)
		return false

	case cpubus.RSYNC:
		// from TIA_HW_Notes.txt:
		//
		// "RSYNC resets the two-phase clock for the HSync counter to the H@1
		// rising edge when strobed."
		tia.PClk = phaseclock.AlignValue

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

		tia.futureRsyncAlign.Schedule(3, 0)
		tia.futureRsyncReset.Schedule(7, 0)
		tia.pendingEvents += 2

		// I've not tested what happens if we reach hsync naturally while the
		// above RSYNC delay is active.

		return false

	case cpubus.HMOVE:
		// the scheduling for HMOVE is divided into two tranches, starting at
		// the same time:
		//
		// the TIA_HW_Notes.txt says this about HMOVE:
		//
		// "It takes 3 CLK after the HMOVE command is received to decode the
		// [SEC] signal (at most 6 CLK depending on the time of STA HMOVE) and
		// a further 4 CLK to set 'more movement required' latches."

		var delayDuration int

		// not forgetting that we count from zero, the following delay
		// values range from 3 to 6, as described in TIA_HW_Notes
		switch tia.PClk {
		case 0:
			delayDuration = 5
		case 1:
			delayDuration = 4
		case 2:
			delayDuration = 4
		case 3:
			delayDuration = 2
		}

		tia.Hmove.FutureLatch.Schedule(delayDuration, 0)
		tia.Hmove.Future.Schedule(delayDuration+3, 0)
		tia.pendingEvents += 2

		// from TIA_HW_Notes:
		//
		// "Also of note, the HMOVE latch used to extend the HBlank time is
		// cleared when the HSync Counter wraps around. This fact is
		// exploited by the trick that involves hitting HMOVE on the 74th
		// CPU cycle of the scanline; the CLK stuffing will still take
		// place during the HBlank and the HSYNC latch will be set just
		// before the counter wraps around. It will then be cleared again
		// immediately (and therefore ignored) when the counter wraps,
		// preventing the HMOVE comb effect."
		//
		// for the this "trick" to work correctly it's important that we get
		// the delay correct for pclk.Count() == 1 above. once that value had
		// been settled the other values fell into place.

		return false
	}

	return true
}

// RSYNCstate returns whether the RSYNC alignment and reset latches are active.
// Both are scheduled at the same time and align takes less time to complete
// than the reset.
func (tia *TIA) RSYNCstate() (bool, bool) {
	return tia.futureRsyncAlign.IsActive(), tia.futureRsyncReset.IsActive()
}

func (tia *TIA) newScanline() {
	// the CPU's WSYNC concludes at the beginning of a scanline
	// from the TIA_1A document:
	//
	// "...WSYNC latch is automatically reset to zero by the
	// leading edge of the next horizontal blank timing signal,
	// releasing the RDY line"
	tia.cpu.SetRDY(true)

	// start HBLANK. start of new scanline for the TIA. turn hblank
	// on
	tia.Hblank = true

	// rather than include the reset signal in the delay, we will
	// manually reset hsync counter when it reaches a count of 57
}

func (tia *TIA) resolveDelayedEvents() {
	// actual vblank signal
	tia.futureVblank.Tick(func(v uint8) {
		tia.pendingEvents--
		tia.sig.VBlank = v&0x02 == 0x02
	})

	tia.futureRsyncAlign.Tick(func(v uint8) {
		tia.pendingEvents--
		tia.newScanline()

		// adjust video elements by the number of visible pixels that have
		// been consumed. adding one to the value because the tv pixel we
		// want to hit has not been reached just yet
		adj := tia.tv.GetCoords().Clock + 1
		if adj > 0 {
			tia.Video.RSYNC(adj)
		}
	})

	tia.futureRsyncReset.Tick(func(v uint8) {
		tia.pendingEvents--
		tia.hsync = polycounter.ResetValue
		tia.PClk = phaseclock.ResetValue
	})

	tia.Hmove.FutureLatch.Tick(func(v uint8) {
		tia.pendingEvents--
		tia.Hmove.Latch = true
	})

	tia.Hmove.Future.Tick(func(v uint8) {
		tia.pendingEvents--
		tia.Video.PrepareSpritesForHMOVE()
		tia.Hmove.ResetRipple()
	})

	tia.futureHsync.Tick(func(v uint8) {
		tia.pendingEvents--
		switch tia.futureHsyncEvent {
		case hsyncEventSHB:
			tia.newScanline()
		case hsyncEventRHS:
			tia.sig.HSync = false
			tia.sig.CBurst = true
		case hsyncEventRCB:
			tia.sig.CBurst = false
		case hsyncEventRHB:
			tia.Hblank = false
		case hsyncEventLRHB:
			tia.Hblank = false
		}
	})
}

// Step moves the state of the tia forward one colour clock. If the state of
// TIA memory has changed then the changes will propogate at the correct time.
// If the state of TIA memory has not changed then execution will be diverted
// to QuickStep().
func (tia *TIA) Step(reg chipbus.ChangedRegister, ct int) {
	tia.ClocksSinceCycle = ct

	// make alterations to video state and playfield
	update := tia.Update(reg)

	// if update has happened we can jump to QuickStep() for the remainder of
	// the Step() process
	if !update {
		tia.QuickStep(ct)
		return
	}

	// tick phase clock
	tia.PClk++
	if tia.PClk >= phaseclock.NumStates {
		tia.PClk = 0
	}

	// "one extra CLK pulse is sent every 4 CLK" and "on every H@1 signal [...]
	// as an extra 'stuffed' clock signal."
	tia.Hmove.Clk = tia.PClk == phaseclock.RisingPhi2

	// tick delayed events and run payload if appropriate
	if tia.pendingEvents > 0 {
		tia.resolveDelayedEvents()
	}

	// tick hsync counter when the Phi2 clock is raised. from TIA_HW_Notes.txt:
	//
	// "This table shows the elapsed number of CLK, CPU cycles, Playfield
	// (PF) bits and Playfield pixels at the start of each counter state
	// (ie when the counter changes to this state on the rising edge of
	// the H@2 clock)."
	//
	// the context of this passage is the Horizontal Sync Counter. It is
	// explicitly saying that the HSYNC counter ticks forward on the rising
	// edge of Phi2.
	if tia.PClk == phaseclock.RisingPhi2 {
		tia.hsync++
		if tia.hsync >= polycounter.LenTable6Bit {
			tia.hsync = 0
		}

		// hsyncDelay is the number of cycles required before, for example, hblank
		// is reset
		const hsyncDelay = 3

		// this switch statement is based on the "Horizontal Sync Counter"
		// table in TIA_HW_Notes.txt. the "key" at the end of that table
		// suggests that (most of) the events are delayed by 4 clocks due to
		// "latching".
		switch tia.hsync {
		case 57:
			// from TIA_HW_Notes.txt:
			//
			// "The HSync counter resets itself after 57 counts; the decode on
			// HCount=56 performs a reset to 000000 delayed by 4 CLK, so
			// HCount=57 becomes HCount=0. This gives a period of 57 counts
			// or 228 CLK."
			tia.hsync = polycounter.ResetValue

			// from TIA_HW_Notes.txt:
			//
			// "Also of note, the HMOVE latch used to extend the HBlank time
			// is cleared when the HSync Counter wraps around. This fact is
			// exploited by the trick that involves hitting HMOVE on the 74th
			// CPU cycle of the scanline; the CLK stuffing will still take
			// place during the HBlank and the HSYNC latch will be set just
			// before the counter wraps around."
			//
			// note that the ripple counter is not reset
			tia.Hmove.Latch = false

		case 56: // [SHB]
			// allow a new scanline event to occur naturally only when an RSYNC
			// has not been scheduled
			if !tia.futureRsyncAlign.IsActive() {
				tia.futureHsyncEvent = hsyncEventSHB
				tia.futureHsync.Schedule(hsyncDelay, 0)
				tia.pendingEvents++
			}

		case 4: // [SHS]
			// start HSYNC. start of new scanline for the television
			// * TIA_HW_Notes.txt does not say there is a 4 clock delay for
			// this. not clear if this is the case.
			//
			// !!TODO: check accuracy of HSync timing
			tia.sig.HSync = true

		case 8: // [RHS]
			// reset HSYNC
			tia.futureHsyncEvent = hsyncEventRHS
			tia.futureHsync.Schedule(hsyncDelay, 0)
			tia.pendingEvents++

		case 12: // [RCB]
			// reset color burst
			tia.futureHsyncEvent = hsyncEventRCB
			tia.futureHsync.Schedule(hsyncDelay, 0)
			tia.pendingEvents++

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
			if !tia.Hmove.Latch {
				tia.futureHsyncEvent = hsyncEventRHB
				tia.futureHsync.Schedule(hsyncDelay, 0)
				tia.pendingEvents++
			}

		// ... and "two counts of the HSync Counter" later ...

		case 18: // [LRHB]
			// late HBLANK off if hmoveLatch is true
			if tia.Hmove.Latch {
				tia.futureHsyncEvent = hsyncEventLRHB
				tia.futureHsync.Schedule(hsyncDelay, 0)
				tia.pendingEvents++
			}
		}
	}

	// mid-subcycle memory resolution
	if update {
		// alter state of video subsystem. occurring after ticking of TIA clock
		// because some the side effects of some registers require that. in
		// particular, the RESxx registers need to have correct information about
		// the state of HBLANK and the HMOVE latch.
		//
		// to see the effect of this, try moving this function call before the
		// HSYNC tick and see how the ball sprite is rendered incorrectly in
		// Keystone Kapers (this is because the ball is reset on the very last
		// pixel and before HBLANK etc. are in the state they need to be)
		update = tia.Video.UpdateSpritePositioning(reg)

		if update {
			// early sprite variations. there is another call later on which
			// also affects the sprite objects
			update = tia.Video.UpdateSpriteVariationsEarly(reg)

			if update {
				// update color registers
				update = tia.Video.UpdateColor(reg)

				if update {
					// update playfield color register (depending on TIA revision)
					if !tia.env.Prefs.Revision.Live.LateColor.Load().(bool) {
						update = tia.Video.UpdatePlayfieldAndBackgroundColor(reg)
					}

					if update {
						// update playfield bits (depending on TIA revisions)
						if !tia.env.Prefs.Revision.Live.LatePFx.Load().(bool) {
							update = tia.Video.UpdatePlayfield(reg)
						}
					}
				}
			}
		}
	}

	// we always call TickSprites but whether or not (and how) the tick
	// actually occurs is left for the sprite object to decide based on the
	// state of the phase clock (isHmove) and the HMOVE ripple count (HmoveCt)
	tia.Video.Tick()

	// update playfield bits (depending on TIA revisions)
	if update {
		if tia.env.Prefs.Revision.Live.LatePFx.Load().(bool) {
			update = tia.Video.UpdatePlayfield(reg)
		}
	}

	// update hmove counter value
	if tia.Hmove.Clk {
		tia.Hmove.Tick()
	}

	// resolve video pixels
	tia.Video.Pixel()
	if tia.Hblank {
		// if hblank is on then we don't sent the resolved color but the
		// zeroblack signal instead
		//
		// we should probably send ZeroBlack in the case of VBLANK but for
		// historic reasons (to do with how we handle debug colours) we leave it
		// up to PixelRenderer implementations to switch to ZeroBlack on VBLANK.
		tia.sig.Color = signal.ZeroBlack
	} else {
		tia.sig.Color = signal.ColorSignal(tia.Video.PixelColor)
	}

	// late memory resolution
	if update {
		if tia.Video.UpdateSpriteHMOVE(reg) {
			if tia.Video.UpdateSpriteVariations(reg) {
				if tia.Video.UpdateSpritePixels(reg) {
					if tia.Audio.ReadMemRegisters(reg) {
						if tia.env.Prefs.Revision.Live.LateColor.Load().(bool) {
							if tia.Video.UpdatePlayfieldAndBackgroundColor(reg) {
								logger.Logf(tia.env, "tia", "memory altered to no affect (%04x=%02x)", reg.Address, reg.Value)
							}
						}
					}
				}
			}
		}
	}

	// mix audio and copy values to television signal
	if ct == 3 && tia.Audio.Step() {
		tia.tv.AudioSignal(signal.AudioSignalAttributes{
			AudioChannel0: tia.Audio.Vol0,
			AudioChannel1: tia.Audio.Vol1,
		})
	}

	// send signal to television
	tia.tv.Signal(tia.sig)
}

// QuickStep ticks the TIA forward one colour clock without checking to see if
// the state of TIA memory has changed.
func (tia *TIA) QuickStep(ct int) {
	tia.ClocksSinceCycle = ct

	// tick phase clock
	tia.PClk++
	if tia.PClk >= phaseclock.NumStates {
		tia.PClk = 0
	}

	// "one extra CLK pulse is sent every 4 CLK" and "on every H@1 signal [...]
	// as an extra 'stuffed' clock signal."
	tia.Hmove.Clk = tia.PClk == phaseclock.RisingPhi2

	// tick delayed events and run payload if appropriate
	if tia.pendingEvents > 0 {
		tia.resolveDelayedEvents()
	}

	// tick hsync counter when the Phi2 clock is raised. from TIA_HW_Notes.txt:
	//
	// "This table shows the elapsed number of CLK, CPU cycles, Playfield
	// (PF) bits and Playfield pixels at the start of each counter state
	// (ie when the counter changes to this state on the rising edge of
	// the H@2 clock)."
	//
	// the context of this passage is the Horizontal Sync Counter. It is
	// explicitly saying that the HSYNC counter ticks forward on the rising
	// edge of Phi2.
	if tia.PClk == phaseclock.RisingPhi2 {
		tia.hsync++
		if tia.hsync >= polycounter.LenTable6Bit {
			tia.hsync = 0
		}

		// hsyncDelay is the number of cycles required before, for example, hblank
		// is reset
		const hsyncDelay = 3

		// this switch statement is based on the "Horizontal Sync Counter"
		// table in TIA_HW_Notes.txt. the "key" at the end of that table
		// suggests that (most of) the events are delayed by 4 clocks due to
		// "latching".
		switch tia.hsync {
		case 57:
			// from TIA_HW_Notes.txt:
			//
			// "The HSync counter resets itself after 57 counts; the decode on
			// HCount=56 performs a reset to 000000 delayed by 4 CLK, so
			// HCount=57 becomes HCount=0. This gives a period of 57 counts
			// or 228 CLK."
			tia.hsync = polycounter.ResetValue

			// from TIA_HW_Notes.txt:
			//
			// "Also of note, the HMOVE latch used to extend the HBlank time
			// is cleared when the HSync Counter wraps around. This fact is
			// exploited by the trick that involves hitting HMOVE on the 74th
			// CPU cycle of the scanline; the CLK stuffing will still take
			// place during the HBlank and the HSYNC latch will be set just
			// before the counter wraps around."
			//
			// note that the ripple counter is not reset
			tia.Hmove.Latch = false

		case 56: // [SHB]
			// allow a new scanline event to occur naturally only when an RSYNC
			// has not been scheduled
			if !tia.futureRsyncAlign.IsActive() {
				tia.futureHsyncEvent = hsyncEventSHB
				tia.futureHsync.Schedule(hsyncDelay, 0)
				tia.pendingEvents++
			}

		case 4: // [SHS]
			// start HSYNC. start of new scanline for the television
			// * TIA_HW_Notes.txt does not say there is a 4 clock delay for
			// this. not clear if this is the case.
			//
			// !!TODO: check accuracy of HSync timing
			tia.sig.HSync = true

		case 8: // [RHS]
			// reset HSYNC
			tia.futureHsyncEvent = hsyncEventRHS
			tia.futureHsync.Schedule(hsyncDelay, 0)
			tia.pendingEvents++

		case 12: // [RCB]
			// reset color burst
			tia.futureHsyncEvent = hsyncEventRCB
			tia.futureHsync.Schedule(hsyncDelay, 0)
			tia.pendingEvents++

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
			if !tia.Hmove.Latch {
				tia.futureHsyncEvent = hsyncEventRHB
				tia.futureHsync.Schedule(hsyncDelay, 0)
				tia.pendingEvents++
			}

		// ... and "two counts of the HSync Counter" later ...

		case 18: // [LRHB]
			// late HBLANK off if hmoveLatch is true
			if tia.Hmove.Latch {
				tia.futureHsyncEvent = hsyncEventLRHB
				tia.futureHsync.Schedule(hsyncDelay, 0)
				tia.pendingEvents++
			}
		}
	}

	// we always call TickSprites but whether or not (and how) the tick
	// actually occurs is left for the sprite object to decide based on the
	// state of the phase clock (isHmove) and the HMOVE ripple count (HmoveCt)
	tia.Video.Tick()

	// update hmove counter value
	if tia.Hmove.Clk {
		tia.Hmove.Tick()
	}

	// resolve video pixels
	tia.Video.Pixel()
	if tia.Hblank {
		// if hblank is on then we don't sent the resolved color but the
		// zeroblack signal instead
		//
		// we should probably send ZeroBlack in the case of VBLANK too but for
		// historic reasons (to do with how we handle debug colours) we leave it
		// up to PixelRenderer implementations to switch to ZeroBlack on VBLANK.
		tia.sig.Color = signal.ZeroBlack
	} else {
		tia.sig.Color = signal.ColorSignal(tia.Video.PixelColor & 0xfe)
	}

	// mix audio and copy values to television signal
	if ct == 3 && tia.Audio.Step() {
		tia.tv.AudioSignal(signal.AudioSignalAttributes{
			AudioChannel0: tia.Audio.Vol0,
			AudioChannel1: tia.Audio.Vol1,
		})
	}

	// send signal to television
	tia.tv.Signal(tia.sig)
}

// AfterPoke implements the chipbus.PokeNotify interface
func (tia *TIA) AfterPoke(reg chipbus.ChangedRegister) {
	_ = tia.Update(reg)
	_ = tia.Video.UpdateSpritePositioning(reg)
	_ = tia.Video.UpdateSpriteVariationsEarly(reg)
	_ = tia.Video.UpdateColor(reg)
	_ = tia.Video.UpdatePlayfieldAndBackgroundColor(reg)
	_ = tia.Video.UpdatePlayfield(reg)
	_ = tia.Video.UpdateSpriteHMOVE(reg)
	_ = tia.Video.UpdateSpriteVariations(reg)
	_ = tia.Video.UpdateSpritePixels(reg)
	_ = tia.Audio.ReadMemRegisters(reg)
}
