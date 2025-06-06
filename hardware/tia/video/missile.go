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

package video

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia/delay"
	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
	"github.com/jetsetilly/gopher2600/hardware/tia/polycounter"
)

// MissileCopies maps missile copies values to descriptions of those values.
var MissileCopies = []string{
	"one copy",
	"two copies [close]",
	"two copies [med]",
	"three copies [close]",
	"two copies [wide]",
	"one copy",
	"three copies [med]",
	"one copy",
}

// MissileSizes maps missile sizes values to descriptions of those values.
var MissileSizes = []string{
	"single width",
	"double width",
	"quadruple width",
	"doubt-quad width",
}

// MissileSprite represents a moveable missile sprite in the VCS graphical display.
// The VCS has two missile sprites.
type MissileSprite struct {
	tia tia

	resetToPlayer func() bool

	// ^^^ references to other parts of the VCS ^^^

	position  polycounter.Polycounter
	pclk      phaseclock.PhaseClock
	MoreHMOVE bool
	Hmove     uint8

	// the following attributes are used for information purposes only:

	label       string
	ResetPixel  int
	HmovedPixel int

	// note whether the last tick was as a result of a HMOVE stuffing tick
	// which left MoreHMOVE in a true state
	lastTickFromHmove bool

	// ^^^ the above are common to all sprite types ^^^
	// (see player sprite for commentary)

	Color         uint8 // equal to missile color
	Enabled       bool
	ResetToPlayer bool

	// for convenience we split the NUSIZ register into size and copies
	Nusiz  uint8
	Size   uint8
	Copies uint8

	// position reset and enclockifier start events are both delayed by a small
	// number of cycles
	futureReset delay.Event
	futureStart delay.Event

	// outputting of pixels is handled by the ball/missile enclockifier.
	// equivalent to the ScanCounter used by the player sprites
	Enclockifier enclockifier

	// state of missile "pixel"
	pixelOn        bool
	pixelCollision bool
}

func newMissileSprite(label string, tia tia, resetToPlayer func() bool) *MissileSprite {
	ms := &MissileSprite{
		tia:           tia,
		resetToPlayer: resetToPlayer,
		label:         label,
	}

	ms.Enclockifier.size = &ms.Size
	ms.position = polycounter.ResetValue

	return ms
}

// Snapshot creates a copy of the missile in its current state.
func (ms *MissileSprite) Snapshot() *MissileSprite {
	n := *ms
	return &n
}

// Plumb changes into missile sprite.
func (ms *MissileSprite) Plumb(tia tia, resetToPlayer func() bool) {
	ms.tia = tia
	ms.resetToPlayer = resetToPlayer
	ms.Enclockifier.size = &ms.Size
}

// Label returns an appropriate name for the sprite.
func (ms *MissileSprite) Label() string {
	return ms.label
}

func (ms *MissileSprite) String() string {
	// the hmove value as maintained by the sprite type is normalised for
	// for purposes of presentation
	normalisedHmove := int(ms.Hmove) - 8
	if normalisedHmove < 0 {
		normalisedHmove = 16 + normalisedHmove
	}

	s := strings.Builder{}
	s.WriteString(ms.label)
	s.WriteString(": ")
	s.WriteString(fmt.Sprintf("%s %s [%03d ", ms.position, ms.pclk, ms.ResetPixel))
	s.WriteString(fmt.Sprintf("> %#1x >", normalisedHmove))
	s.WriteString(fmt.Sprintf(" %03d", ms.HmovedPixel))
	if ms.MoreHMOVE {
		s.WriteString("*] ")
	} else {
		s.WriteString("] ")
	}

	// interpret size and copies values
	switch ms.Copies {
	case 0x0:
		s.WriteString("one copy")
	case 0x1:
		s.WriteString("two copies [close]")
	case 0x2:
		s.WriteString("two copies [med]")
	case 0x3:
		s.WriteString("three copies [close]")
	case 0x4:
		s.WriteString("two copies [wide]")
	case 0x5:
		s.WriteString("one copy")
	case 0x6:
		s.WriteString("three copies [med]")
	case 0x7:
		s.WriteString("one copy")
	default:
		panic("illegal copies value for missile")
	}

	switch ms.Size {
	case 0x0:
	case 0x1:
		s.WriteString(" 2x")
	case 0x2:
		s.WriteString(" 4x")
	case 0x3:
		s.WriteString(" 8x")
	default:
		panic("illegal size value for missile")
	}

	if ms.MoreHMOVE {
		s.WriteString(fmt.Sprintf(" hmoving [%04b],", ms.Hmove))
	}

	if ms.Enclockifier.Active {
		s.WriteString(fmt.Sprintf(" drw %s,", ms.Enclockifier.String()))
	}

	if !ms.Enabled {
		s.WriteString(" disb,")
	}

	if ms.ResetToPlayer {
		s.WriteString(" >pl<,")
	}

	return strings.TrimSuffix(s.String(), ",")
}

func (ms *MissileSprite) rsync(adjustment int) {
	ms.ResetPixel -= adjustment
	ms.HmovedPixel -= adjustment
	if ms.ResetPixel < 0 {
		ms.ResetPixel += specification.ClksVisible
	}
	if ms.HmovedPixel < 0 {
		ms.HmovedPixel += specification.ClksVisible
	}
}

func (ms *MissileSprite) tickHBLANK() bool {
	// check to see if there is more movement required for this sprite
	ms.MoreHMOVE = ms.MoreHMOVE && compareHMOVE(ms.tia.hmove.Ripple, ms.Hmove)
	if !ms.MoreHMOVE {
		return false
	}

	// update hmoved pixel value & adjust for screen boundary
	ms.HmovedPixel--
	if ms.HmovedPixel < 0 {
		ms.HmovedPixel += specification.ClksVisible
	}

	ms.lastTickFromHmove = ms.MoreHMOVE

	return ms.tick()
}

func (ms *MissileSprite) tickHMOVE() bool {
	// check to see if there is more movement required for this sprite
	ms.MoreHMOVE = ms.MoreHMOVE && compareHMOVE(ms.tia.hmove.Ripple, ms.Hmove)

	// cancel motion clock if necessary
	if ms.MoreHMOVE && ms.tia.env.Prefs.Revision.Live.LostMOTCK.Load().(bool) {
		return false
	}

	ms.lastTickFromHmove = ms.MoreHMOVE

	return ms.tick()
}

// returns true if pixel has changed.
func (ms *MissileSprite) tick() bool {
	// reset-to-player placement note: we don't do the missile-to-player reset
	// unless we're hmoving or ticking. if we place this block before the
	// "early return if nothing to do" block above, then it will produce
	// incorrect results. we can see this (occasionally) in Supercharger
	// Frogger - the top row of trucks will sometimes extend by a pixel as they
	// drive off screen.
	if ms.ResetToPlayer && ms.resetToPlayer() {
		ms.position = polycounter.ResetValue
		ms.pclk = phaseclock.ResetValue

		// missile-to-player also resets position information
		ms.ResetPixel = ms.tia.tv.GetCoords().Clock
		ms.HmovedPixel = ms.ResetPixel
	}

	ms.pclk++
	if ms.pclk >= phaseclock.NumStates {
		ms.pclk = 0
	}

	if ms.pclk == phaseclock.RisingPhi2 {
		ms.position++
		if ms.position >= polycounter.LenTable6Bit {
			ms.position = 0
		}

		// start drawing if there is no reset or it has just started AND
		// there wasn't a reset event ongoing when the current event
		// started
		if !ms.futureReset.IsActive() || ms.futureReset.JustStarted() {
			switch ms.position {
			case 3:
				if ms.Copies == 0x01 || ms.Copies == 0x03 {
					ms.futureStart.Schedule(4, 1)
				}
			case 7:
				if ms.Copies == 0x03 || ms.Copies == 0x02 || ms.Copies == 0x06 {
					cpy := 1
					if ms.Copies == 0x03 {
						cpy = 2
					}
					ms.futureStart.Schedule(4, uint8(cpy))
				}
			case 15:
				if ms.Copies == 0x04 || ms.Copies == 0x06 {
					cpy := 1
					if ms.Copies == 0x06 {
						cpy = 2
					}
					ms.futureStart.Schedule(4, uint8(cpy))
				}
			case 39:
				ms.futureStart.Schedule(4, 0)
			case 40:
				ms.position = polycounter.ResetValue
			}
		}
	}

	ms.Enclockifier.tick()

	// tick delayed events. note that the order of these ticks is important.
	ms.futureReset.Tick(ms._futureResetPosition)
	ms.futureStart.Tick(ms._futureStartDrawingEvent)

	return true
}

func (ms *MissileSprite) _futureStartDrawingEvent(v uint8) {
	ms.Enclockifier.Cpy = int(v)
	ms.Enclockifier.start()
}

func (ms *MissileSprite) prepareForHMOVE() {
	ms.MoreHMOVE = true

	if *ms.tia.hblank {
		// adjust hmovedPixel value. this value is subject to further change so
		// long as moreHMOVE is true. the String() function this value is
		// annotated with a "*" to indicate that HMOVE is still in progress
		ms.HmovedPixel += 8

		// adjust for screen boundary
		if ms.HmovedPixel > specification.ClksVisible {
			ms.HmovedPixel -= specification.ClksVisible
		}
	}
}

func (ms *MissileSprite) resetPosition() {
	// see player sprite resetPosition() for commentary on delay values
	delay := 4
	if *ms.tia.hblank {
		// the condition to differentiate a delay of 2 and three is taken from
		// the ball sprite. there are better tests for the ball sprite
		if ms.tia.hmove.Ripple > 0 {
			delay = 2
		} else {
			delay = 3
		}
	}

	// drawing of missile sprite is paused and will resume upon reset
	// completion. compare to ball sprite where drawing is ended and then
	// re-started under all conditions
	//
	// important to note we only pause if the draw/start events are not about
	// to end. in other words, if they are not about to end they are allowed to
	// continue naturally while reset event is waiting to conclude
	if !ms.Enclockifier.aboutToEnd() {
		ms.Enclockifier.Paused = true
	}
	if ms.futureStart.IsActive() && !ms.futureStart.AboutToEnd() {
		ms.futureStart.Pause()
	}

	// stop any existing reset events. generally, this codepath will not apply
	// because a resetPositionEvent will conclude before being triggered again.
	// but it is possible when using a very quick instruction on the reset register,
	// like a zero page INC, for requests to overlap
	//
	// in the case of the missile sprite, we can see such an occurrence in the
	// test.bin test ROM
	if ms.futureReset.IsActive() {
		ms.futureReset.Push()
		return
	}

	ms.futureReset.Schedule(delay, 0)
}

func (ms *MissileSprite) _futureResetPosition(_ uint8) {
	// the pixel at which the sprite has been reset, in relation to the
	// left edge of the screen
	ms.ResetPixel = ms.tia.tv.GetCoords().Clock

	if ms.ResetPixel >= 0 {
		// resetPixel adjusted by 1 because the tv is not yet in the correct
		// position
		ms.ResetPixel++

		// adjust resetPixel for screen boundaries
		if ms.ResetPixel > specification.ClksVisible {
			ms.ResetPixel -= specification.ClksVisible
		}

		// by definition the current pixel is the same as the reset pixel at
		// the moment of reset
		ms.HmovedPixel = ms.ResetPixel
	} else {
		// if reset occurs off-screen then force reset pixel to be zero
		// (see commentary in ball sprite for detailed reasoning of this
		// branch)
		ms.ResetPixel = 0
		ms.HmovedPixel = 7
	}

	// reset both sprite position and clock
	ms.position = polycounter.ResetValue
	ms.pclk = phaseclock.ResetValue

	ms.Enclockifier.force()
	if ms.futureStart.IsActive() {
		v := ms.futureStart.Force()
		ms._futureStartDrawingEvent(v)
	}
}

func (ms *MissileSprite) setResetToPlayer(on bool) {
	ms.ResetToPlayer = on
}

func (ms *MissileSprite) pixel() {
	if !ms.Enabled {
		ms.pixelOn = false
		ms.pixelCollision = *ms.tia.hblank && ms.Enabled
		return
	}

	// the missile sprite has a special state where a stuffed HMOVE clock
	// forces the draw signal to true *if* the enclockifier is to begin next
	// cycle.
	earlyStart := ms.lastTickFromHmove && ms.futureStart.AboutToEnd()

	// similarly, in the event of a stuffed HMOVE clock, and when the
	// enclockifier is about to produce its last pixel
	earlyEnd := ms.pclk != phaseclock.FallingPhi1 && ms.lastTickFromHmove && ms.Enclockifier.aboutToEnd()

	// see ball sprite for explanation for the LatePhi1() condition

	// both earlyStart and earlyEnd conditions are fully explained in the
	// AtariAge post "Cosmic Ark Star Field Revisited" by crispy. as suggested
	// by the post title this is the key to implementing the starfield in the
	// Cosmic Ark ROM

	// whether a pixel is output also depends on whether resetToPlayer is off
	ms.pixelOn = !ms.ResetToPlayer && !earlyEnd && (ms.Enclockifier.Active || earlyStart)

	ms.pixelCollision = ms.pixelOn || (*ms.tia.hblank && ms.Enabled && ms.futureStart.AboutToEnd())

	// reset lastTickFromHmove flag
	ms.lastTickFromHmove = false
}

func (ms *MissileSprite) setEnable(enable bool) {
	ms.Enabled = enable
}

// SetNUSIZ is called when the NUSIZ register changes. It should also be used
// to set the NUSIZ value from a debugger for immediate effect.
func (ms *MissileSprite) SetNUSIZ(value uint8) {
	// note raw NUSIZ value
	ms.Nusiz = value

	// for convenience we pick out the size and count values from the NUSIZ
	// value
	ms.Size = (value & 0x30) >> 4
	ms.Copies = value & 0x07
}

func (ms *MissileSprite) setColor(value uint8) {
	ms.Color = value
}

func (ms *MissileSprite) setHmoveValue(v uint8) {
	ms.Hmove = (v ^ 0x80) >> 4
}

func (ms *MissileSprite) clearHmoveValue() {
	ms.Hmove = 0x08
}
