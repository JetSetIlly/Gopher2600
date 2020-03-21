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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package tia

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/input"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/hardware/tia/future"
	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
	"github.com/jetsetilly/gopher2600/hardware/tia/polycounter"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
	"github.com/jetsetilly/gopher2600/television"
)

// TIA contains all the sub-components of the VCS TIA sub-system
type TIA struct {
	tv         television.Television
	mem        bus.ChipBus
	vblankBits *input.VBlankBits

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

	// - hmoveCt counts backwards from 15 to -1 (represented by 255). note that
	// unlike how it is described in TIA_HW_Notes.txt, we always send the extra
	// tick to the sprites on Phi1.  however, we also send the hmoveCt value,
	// whether or not the extra should be honoured is up to the sprite.
	// (TIA_HW_Notes.txt says that hmoveCt is checked *before* sending the
	// extra tick)
	hmoveCt uint8

	// TIA_HW_Notes.txt describes the hsync counter:
	//
	// "The HSync counter counts from 0 to 56 once for every TV scan-line
	// before wrapping around, a period of 57 counts at 1/4 CLK (57*4=228 CLK).
	// The counter decodes shown below provide all the horizontal timing for
	// the control lines used to construct a valid TV signal."
	hsync *polycounter.Polycounter
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

// Label returns an identifying label for the TIA
func (tia TIA) Label() string {
	return "TIA"
}

func (tia TIA) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s %s %03d %04.01f",
		tia.hsync, tia.pclk,
		tia.videoCycles, float64(tia.videoCycles)/3.0,
	))

	if tia.hmoveCt != 0xff {
		s.WriteString(fmt.Sprintf(" hm=%04b", tia.hmoveCt))
	}

	return s.String()
}

// NewTIA creates a TIA, to be used in a VCS emulation
func NewTIA(tv television.Television, mem bus.ChipBus, vblankBits *input.VBlankBits) (*TIA, error) {
	tia := TIA{
		tv:         tv,
		mem:        mem,
		vblankBits: vblankBits,
		hblank:     true}

	var err error

	tia.hsync, err = polycounter.New(6)
	if err != nil {
		return nil, err
	}

	tia.Delay = future.NewTicker("TIA")

	tia.pclk.Reset()
	tia.hmoveCt = 0xff

	tia.Video, err = video.NewVideo(mem, &tia.pclk, tia.hsync, tv, &tia.hblank, &tia.hmoveLatch)
	if err != nil {
		return nil, err
	}

	tia.Audio = audio.NewAudio()
	if err != nil {
		return nil, err
	}

	return &tia, nil
}

// UpdateTIA checks for side effects in the TIA sub-system.
//
// Returns true if ChipData has *not* been serviced.
func (tia *TIA) UpdateTIA(data bus.ChipData) bool {
	switch data.Name {
	case "VSYNC":
		tia.sig.VSync = data.Value&0x02 == 0x02
		return false

	case "VBLANK":
		// homebrew Donkey Kong shows the need for a delay of at least one
		// cycle for VBLANK. see area just before score box on play screen
		tia.Delay.ScheduleWithArg(1, tia._futureVBLANK, data.Value, "VBLANK")

		return false

	case "WSYNC":
		// CPU has indicated that it wants to wait for the beginning of the
		// next scanline. value is reset to false when TIA reaches end of
		// scanline
		tia.wsync = true
		return false

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

		tia.rsyncEvent = tia.Delay.Schedule(3, tia._futureRSYNCnewScanline, "RSYNC (new scanline)")

		// I've not test what happens if we reach hsync naturally while the
		// above RSYNC delay is active.

		return false

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
		// values range from 3 to 6, as described in TIA_HW_Notes
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

		tia.Delay.Schedule(delay, tia._futureHMOVElatch, "HMOVE")

		delay += 3
		tia.hmoveEvent = tia.Delay.Schedule(delay, tia._futureHMOVEprep, "HMOVE (prep)")

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

		return false
	}

	return true
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

func (tia *TIA) _futureVBLANK(v interface{}) {
	// actual vblank signal
	tia.sig.VBlank = v.(uint8)&0x02 == 0x02

	// dump paddle capacitors to ground
	tia.vblankBits.SetGroundPaddles(v.(uint8)&0x80 == 0x80)

	// joystick fire button latches
	tia.vblankBits.SetLatchFireButton(v.(uint8)&0x40 == 0x40)
}

func (tia *TIA) _futureRSYNCreset() {
	tia.hsync.Reset()
	tia.pclk.Reset()
	tia.rsyncEvent = nil
}

func (tia *TIA) _futureRSYNCnewScanline() {
	tia.newScanline()

	// adjust video elements by the number of visible pixels that have
	// been consumed. adding one to the value because the tv pixel we
	// want to hit has not been reached just yet
	adj, _ := tia.tv.GetState(television.ReqHorizPos)
	adj++
	if adj > 0 {
		tia.Video.RSYNC(adj)
	}

	tia.rsyncEvent = tia.Delay.Schedule(4, tia._futureRSYNCreset, "RSYNC (reset)")
}

func (tia *TIA) _futureHMOVElatch() {
	tia.hmoveLatch = true
}

func (tia *TIA) _futureHMOVEprep() {
	tia.Video.PrepareSpritesForHMOVE()
	tia.hmoveCt = 15
	tia.hmoveEvent = nil
}

func (tia *TIA) _futureResetHSYNC() {
	tia.sig.HSync = false
	tia.sig.CBurst = true
}

func (tia *TIA) _futureResetColorBurst() {
	tia.sig.CBurst = false
}

func (tia *TIA) _futureResetHBlank() {
	tia.hblank = false
}
