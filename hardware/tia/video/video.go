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
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/tia/delay"
	"github.com/jetsetilly/gopher2600/hardware/tia/hmove"
	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
	"github.com/jetsetilly/gopher2600/hardware/tia/polycounter"
)

// Element is used to record from which video sub-system the pixel
// was generated, taking video priority into account.
type Element int

// List of valid Element Signals.
const (
	ElementBackground Element = iota
	ElementBall
	ElementPlayfield
	ElementPlayer0
	ElementPlayer1
	ElementMissile0
	ElementMissile1
)

func (e Element) String() string {
	switch e {
	case ElementBackground:
		return "Background"
	case ElementBall:
		return "Ball"
	case ElementPlayfield:
		return "Playfield"
	case ElementPlayer0:
		return "Player 0"
	case ElementPlayer1:
		return "Player 1"
	case ElementMissile0:
		return "Missile 0"
	case ElementMissile1:
		return "Missile 1"
	}
	panic("unknown video element")
}

// TV defines the television functions required by the Video type(s).
type TV interface {
	GetCoords() coords.TelevisionCoords
}

// Video contains all the components of the video sub-system of the VCS TIA chip.
type Video struct {
	// reference to important TIA state
	tia tia

	// collision matrix
	Collisions *Collisions

	// playfield
	Playfield *Playfield

	// sprite objects
	Player0  *PlayerSprite
	Player1  *PlayerSprite
	Missile0 *MissileSprite
	Missile1 *MissileSprite
	Ball     *BallSprite

	// LastElement records from which TIA video sub-system the most recent
	// pixel was generated, taking priority into account. see Pixel() function
	// for details
	LastElement Element

	// keep track of whether any tia element has changed since last colour clock
	//
	// how frequently we reset tiaHasChanged to true is the key to optimisation
	// of the TIA. extreme care should be taken when making a decisions
	// relating to this variable
	tiaHasChanged bool

	// color of Video output
	PixelColor uint8

	// some register writes require a small latching delay. they never overlap
	// so one event is sufficient
	writing         delay.Event
	writingRegister cpubus.Register
}

// tia is a convenient packaging of TIA state that is required by the playfield/sprites.
type tia struct {
	env    *environment.Environment
	tv     TV
	pclk   *phaseclock.PhaseClock
	hsync  *polycounter.Polycounter
	hblank *bool
	hmove  *hmove.Hmove
}

// NewVideo is the preferred method of initialisation for the Video sub-system.
//
// The playfield type requires access access to the TIA's phaseclock and
// polyucounter and is used to decide which part of the playfield is to be
// drawn.
//
// The sprites meanwhile require access to the television. This is for
// generating information about the sprites reset position - a debugging only
// requirement but of no performance related consequeunce.
//
// The references to the TIA's HBLANK state and whether HMOVE is latched, are
// required to tune the delays experienced by the various sprite events (eg.
// reset position).
func NewVideo(env *environment.Environment, mem chipbus.Memory, tv TV, pclk *phaseclock.PhaseClock,
	hsync *polycounter.Polycounter, hblank *bool, hmove *hmove.Hmove) *Video {
	tia := tia{
		env:    env,
		tv:     tv,
		pclk:   pclk,
		hsync:  hsync,
		hblank: hblank,
		hmove:  hmove,
	}

	vd := Video{
		tia:        tia,
		Collisions: newCollisions(mem),
		Playfield:  newPlayfield(tia),
		Player0:    newPlayerSprite("Player 0", tia),
		Player1:    newPlayerSprite("Player 1", tia),
		Ball:       newBallSprite("Ball", tia),
	}

	vd.Missile0 = newMissileSprite("Missile 0", tia, vd.Player0.triggerMissileReset)
	vd.Missile1 = newMissileSprite("Missile 1", tia, vd.Player1.triggerMissileReset)

	return &vd
}

// Snapshot creates a copy of the Video sub-system in its current state.
func (vd *Video) Snapshot() *Video {
	n := *vd
	n.Collisions = vd.Collisions.Snapshot()
	n.Playfield = vd.Playfield.Snapshot()
	n.Player0 = vd.Player0.Snapshot()
	n.Player1 = vd.Player1.Snapshot()
	n.Missile0 = vd.Missile0.Snapshot()
	n.Missile1 = vd.Missile1.Snapshot()
	n.Ball = vd.Ball.Snapshot()
	return &n
}

// Plumb ChipBus into TIA/Video components. Update pointers that refer to parent TIA.
func (vd *Video) Plumb(env *environment.Environment, mem chipbus.Memory, tv TV, pclk *phaseclock.PhaseClock,
	hsync *polycounter.Polycounter, hblank *bool, hmove *hmove.Hmove) {
	vd.Collisions.Plumb(mem)

	vd.tia = tia{
		env:    env,
		tv:     tv,
		pclk:   pclk,
		hsync:  hsync,
		hblank: hblank,
		hmove:  hmove,
	}

	vd.Playfield.Plumb(vd.tia)
	vd.Player0.Plumb(vd.tia)
	vd.Player1.Plumb(vd.tia)
	vd.Missile0.Plumb(vd.tia, vd.Player0.triggerMissileReset)
	vd.Missile1.Plumb(vd.tia, vd.Player1.triggerMissileReset)
	vd.Ball.Plumb(vd.tia)
}

// RSYNC adjusts the debugging information of the sprites when an RSYNC is
// triggered.
func (vd *Video) RSYNC(adjustment int) {
	vd.Player0.rsync(adjustment)
	vd.Player1.rsync(adjustment)
	vd.Missile0.rsync(adjustment)
	vd.Missile1.rsync(adjustment)
	vd.Ball.rsync(adjustment)
}

// Tick moves all video elements forward one video cycle. This is the
// conceptual equivalent of the hardware MOTCK line.
func (vd *Video) Tick() {
	vd.writing.Tick(func(v uint8) {
		switch vd.writingRegister {
		case cpubus.PF0:
			vd.Playfield.setPF0(v)
		case cpubus.PF1:
			vd.Playfield.setPF1(v)
		case cpubus.PF2:
			vd.Playfield.setPF2(v)
		case cpubus.HMP0:
			vd.Player0.setHmoveValue(v)
		case cpubus.HMP1:
			vd.Player1.setHmoveValue(v)
		case cpubus.HMM0:
			vd.Missile0.setHmoveValue(v)
		case cpubus.HMM1:
			vd.Missile1.setHmoveValue(v)
		case cpubus.HMBL:
			vd.Ball.setHmoveValue(v)
		case cpubus.HMCLR:
			vd.Player0.clearHmoveValue()
			vd.Player1.clearHmoveValue()
			vd.Missile0.clearHmoveValue()
			vd.Missile1.clearHmoveValue()
			vd.Ball.clearHmoveValue()

		// these registers will only ever be pushed onto the writing queue if
		// the TIA revisison is set accordingly. normally, GRP0 and GRP1 are
		// set without delay.
		case cpubus.GRP0:
			vd.Player1.setOldGfxData()
		case cpubus.GRP1:
			vd.Player0.setOldGfxData()
			vd.Ball.setEnableDelay()
		}
	})

	// playfield must tick every time regardless of hblank or hmove state
	vd.tiaHasChanged = vd.Playfield.tick() || vd.tiaHasChanged

	// ticking of sprites can be more selective
	if *vd.tia.hblank {
		if vd.tia.hmove.Clk {
			// we can check the state of MoreHMOVE for each sprite before
			// calling tickHBLANK() because sprites are *only* ticked by the
			// HMOVE Clk when HBLANK is active - and if MoreHMOVE is false
			// there is nothing else to do
			vd.tiaHasChanged = (vd.Player0.MoreHMOVE && vd.Player0.tickHBLANK()) || vd.tiaHasChanged
			vd.tiaHasChanged = (vd.Player1.MoreHMOVE && vd.Player1.tickHBLANK()) || vd.tiaHasChanged
			vd.tiaHasChanged = (vd.Missile0.MoreHMOVE && vd.Missile0.tickHBLANK()) || vd.tiaHasChanged
			vd.tiaHasChanged = (vd.Missile1.MoreHMOVE && vd.Missile1.tickHBLANK()) || vd.tiaHasChanged
			vd.tiaHasChanged = (vd.Ball.MoreHMOVE && vd.Ball.tickHBLANK()) || vd.tiaHasChanged
		}
	} else if vd.tia.hmove.Clk {
		vd.tiaHasChanged = vd.Player0.tickHMOVE() || vd.tiaHasChanged
		vd.tiaHasChanged = vd.Player1.tickHMOVE() || vd.tiaHasChanged
		vd.tiaHasChanged = vd.Missile0.tickHMOVE() || vd.tiaHasChanged
		vd.tiaHasChanged = vd.Missile1.tickHMOVE() || vd.tiaHasChanged
		vd.tiaHasChanged = vd.Ball.tickHMOVE() || vd.tiaHasChanged
	} else {
		vd.tiaHasChanged = vd.Player0.tick() || vd.tiaHasChanged
		vd.tiaHasChanged = vd.Player1.tick() || vd.tiaHasChanged
		vd.tiaHasChanged = vd.Missile0.tick() || vd.tiaHasChanged
		vd.tiaHasChanged = vd.Missile1.tick() || vd.tiaHasChanged
		vd.tiaHasChanged = vd.Ball.tick() || vd.tiaHasChanged
	}
}

// PrepareSpritesForHMOVE should be called whenever HMOVE is triggered.
func (vd *Video) PrepareSpritesForHMOVE() {
	vd.Player0.prepareForHMOVE()
	vd.Player1.prepareForHMOVE()
	vd.Missile0.prepareForHMOVE()
	vd.Missile1.prepareForHMOVE()
	vd.Ball.prepareForHMOVE()
}

// Pixel returns the color of the pixel at the current clock and also sets the
// collision registers. It will default to returning the background color if no
// sprite or playfield pixel is present.
func (vd *Video) Pixel() {
	// if nothing has changed since last pixel then return early and leave the
	// Video.PixelColor at the same value
	if !vd.tiaHasChanged {
		return
	}
	vd.tiaHasChanged = false

	// update pixel information of sprites. the pixel of the playfield is an
	// implicit result of the tick() function
	vd.Player0.pixel()
	vd.Player1.pixel()
	vd.Missile0.pixel()
	vd.Missile1.pixel()
	vd.Ball.pixel()

	// only check for collisions if at least one sprite thinks it might be
	// worth doing
	if vd.Player0.pixelCollision || vd.Player1.pixelCollision ||
		vd.Missile0.pixelCollision || vd.Missile1.pixelCollision ||
		vd.Ball.pixelCollision {

		vd.Collisions.tick(vd.Player0.pixelCollision, vd.Player1.pixelCollision,
			vd.Missile0.pixelCollision, vd.Missile1.pixelCollision,
			vd.Ball.pixelCollision, vd.Playfield.colorLatch)
	} else {
		vd.Collisions.LastColorClock.reset()
	}

	// prioritisation of pixels:
	//
	// there have been bugs in earlier versions of this code regarding the
	// priority of the ball sprite in scoremode. the following code is correct
	// and satisifies all known test cases.
	//
	// note the technical description from Supercat on AtariAge.
	//
	// "To be hyper-precise, Score Mode causes the player 0/1 color circuits to
	// be activated anyplace the playfield is active. When sprites have
	// priority over playfield, this "covers up" the playfield color there. If
	// playfield priority is enabled, the activated 0/1 colors get overruled by
	// the playfield color, rendering score mode ineffective. Note that in
	// score mode, the left half of the playfield has priority over the
	// player/missile 1 sprites."
	//
	// https://atariage.com/forums/topic/166193-playfield-score-mode-effect-on-ball/?tab=comments#comment-2083030
	//
	// My misunderstanding was caused by changing the priority of the ball
	// sprite when the priority bit was on alongside the scoremode bit.
	//
	if vd.Playfield.Priority { // priority take precedence of scoremode
		if vd.Playfield.colorLatch { // priority 1
			vd.PixelColor = vd.Playfield.color
			vd.LastElement = ElementPlayfield
		} else if vd.Ball.pixelOn { // priority 1 (ball is same color as playfield)
			vd.PixelColor = vd.Ball.Color
			vd.LastElement = ElementBall
		} else if vd.Player0.pixelOn { // priority 2
			vd.PixelColor = vd.Player0.Color
			vd.LastElement = ElementPlayer0
		} else if vd.Missile0.pixelOn { // priority 2 (missile 0 is same color as player 0)
			vd.PixelColor = vd.Missile0.Color
			vd.LastElement = ElementMissile0
		} else if vd.Player1.pixelOn { // priority 3
			vd.PixelColor = vd.Player1.Color
			vd.LastElement = ElementPlayer1
		} else if vd.Missile1.pixelOn { // priority 3 (missile 1 is same color as player 1)
			vd.PixelColor = vd.Missile1.Color
			vd.LastElement = ElementMissile1
		} else {
			vd.PixelColor = vd.Playfield.BackgroundColor
			vd.LastElement = ElementBackground
		}
	} else if vd.Playfield.Scoremode { // scoremode applies when priority bit os not set
		switch vd.Playfield.Region {
		case RegionOffScreen:
			fallthrough
		case RegionLeft:
			if vd.Playfield.colorLatch { // priority 1 (playfield takes color of player 0)
				vd.PixelColor = vd.Player0.Color
				vd.LastElement = ElementPlayfield
			} else if vd.Player0.pixelOn { // priority 1 (same color as playfield)
				vd.PixelColor = vd.Player0.Color
				vd.LastElement = ElementPlayer0
			} else if vd.Missile0.pixelOn { // priority 1 same color as playfield)
				vd.PixelColor = vd.Missile0.Color
				vd.LastElement = ElementMissile0
			} else if vd.Player1.pixelOn { // priority 2
				vd.PixelColor = vd.Player1.Color
				vd.LastElement = ElementPlayer1
			} else if vd.Missile1.pixelOn { // priority 2 (missile 1 is same color as player 1)
				vd.PixelColor = vd.Missile1.Color
				vd.LastElement = ElementMissile1
			} else if vd.Ball.pixelOn { // priority 3
				vd.PixelColor = vd.Ball.Color
				vd.LastElement = ElementBall
			} else {
				vd.PixelColor = vd.Playfield.BackgroundColor
				vd.LastElement = ElementBackground
			}
		case RegionRight:
			if vd.Player0.pixelOn { // priority 1
				vd.PixelColor = vd.Player0.Color
				vd.LastElement = ElementPlayer0
			} else if vd.Missile0.pixelOn { // priority 1 (missile 0 is same colour as player 0)
				vd.PixelColor = vd.Missile0.Color
				vd.LastElement = ElementMissile0
			} else if vd.Player1.pixelOn { // priority 2
				vd.PixelColor = vd.Player1.Color
				vd.LastElement = ElementPlayer1
			} else if vd.Missile1.pixelOn { // priority 2 (missile 1 is same colour as player 1)
				vd.PixelColor = vd.Missile1.Color
				vd.LastElement = ElementMissile1
			} else if vd.Playfield.colorLatch { // priority 2 (playfield takes color of player 1)
				vd.PixelColor = vd.Player1.Color
				vd.LastElement = ElementPlayfield
			} else if vd.Ball.pixelOn { // priority 3
				vd.PixelColor = vd.Ball.Color
				vd.LastElement = ElementBall
			} else {
				vd.PixelColor = vd.Playfield.BackgroundColor
				vd.LastElement = ElementBackground
			}
		}
	} else { // normal priority
		if vd.Player0.pixelOn { // priority 1
			vd.PixelColor = vd.Player0.Color
			vd.LastElement = ElementPlayer0
		} else if vd.Missile0.pixelOn { // priority 1 (missile 0 is same color as player 0)
			vd.PixelColor = vd.Missile0.Color
			vd.LastElement = ElementMissile0
		} else if vd.Player1.pixelOn { // priority 2
			vd.PixelColor = vd.Player1.Color
			vd.LastElement = ElementPlayer1
		} else if vd.Missile1.pixelOn { // priority 2 (missile 1 is same color as player 1)
			vd.PixelColor = vd.Missile1.Color
			vd.LastElement = ElementMissile1
		} else if vd.Ball.pixelOn { // priority 3
			vd.PixelColor = vd.Ball.Color
			vd.LastElement = ElementBall
		} else if vd.Playfield.colorLatch { // priority 3 (playfield is same color as ball)
			vd.PixelColor = vd.Playfield.color
			vd.LastElement = ElementPlayfield
		} else {
			vd.PixelColor = vd.Playfield.BackgroundColor
			vd.LastElement = ElementBackground
		}
	}
}

// UpdatePlayfield checks TIA memory for new playfield data. Note that CTRLPF
// is serviced in UpdateSpriteVariations().
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdatePlayfield(data chipbus.ChangedRegister) bool {
	// homebrew Donkey Kong shows the need for a delay of at least two cycles
	// to write new playfield data
	switch data.Register {
	case cpubus.PF0:
		vd.writingRegister = data.Register
		vd.writing.Schedule(2, data.Value)
	case cpubus.PF1:
		vd.writingRegister = data.Register
		vd.writing.Schedule(2, data.Value)
	case cpubus.PF2:
		vd.writingRegister = data.Register
		vd.writing.Schedule(2, data.Value)
	default:
		return true
	}

	return false
}

// UpdateSpriteHMOVE checks TIA memory for changes in sprite HMOVE settings.
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdateSpriteHMOVE(data chipbus.ChangedRegister) bool {
	switch data.Register {
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

	// there is no information about whether response to HMOVE value changes
	// are immediate or take effect after a short delay. experimentation
	// reveals that a delay is required. the reasoning for the value is as
	// below:
	//
	// delay of at least zero (1 additiona cycle) is required. we can see this
	// in the Midnight Magic ROM where the left gutter separator requires it
	//
	// a delay too high (3 or higher) causes the barber pole test ROM to fail
	//
	// (19/01/20) a delay of anything other than 0 or 1, causes Panda Chase to
	// fail.
	//
	// (28/01/20) a delay of anything lower than 1, causes the text in the
	// BASIC ROM to fail
	//
	// the only common value that satisfies all test cases is 1, which equates
	// to a delay of two cycles
	case cpubus.HMP0:
		vd.writingRegister = data.Register
		vd.writing.Schedule(1, data.Value&HMxxMask)
	case cpubus.HMP1:
		vd.writingRegister = data.Register
		vd.writing.Schedule(1, data.Value&HMxxMask)
	case cpubus.HMM0:
		vd.writingRegister = data.Register
		vd.writing.Schedule(1, data.Value&HMxxMask)
	case cpubus.HMM1:
		vd.writingRegister = data.Register
		vd.writing.Schedule(1, data.Value&HMxxMask)
	case cpubus.HMBL:
		vd.writingRegister = data.Register
		vd.writing.Schedule(1, data.Value&HMxxMask)
	case cpubus.HMCLR:
		vd.writingRegister = data.Register
		vd.writing.Schedule(1, 0)

	default:
		return true
	}

	vd.tiaHasChanged = true
	return false
}

// UpdateSpritePositioning checks TIA memory for strobing of reset registers.
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdateSpritePositioning(data chipbus.ChangedRegister) bool {
	switch data.Register {
	// the reset registers *must* be serviced after HSYNC has been ticked.
	// resets are resolved after a short delay, governed by the sprite itself
	case cpubus.RESP0:
		vd.Player0.resetPosition()
	case cpubus.RESP1:
		vd.Player1.resetPosition()
	case cpubus.RESM0:
		vd.Missile0.resetPosition()
	case cpubus.RESM1:
		vd.Missile1.resetPosition()
	case cpubus.RESBL:
		vd.Ball.resetPosition()
	case cpubus.VDELBL:
		vd.Ball.setVerticalDelay(data.Value&0x01 == 0x01)
	default:
		return true
	}

	vd.tiaHasChanged = true
	return false
}

// UpdateColor checks TIA memory for changes to color registers.
//
// See UpdatePlayfieldAndBackgroundColor() also.
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdateColor(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.COLUP0:
		vd.Player0.setColor(data.Value & 0xfe)
		vd.Missile0.setColor(data.Value & 0xfe)
	case cpubus.COLUP1:
		vd.Player1.setColor(data.Value & 0xfe)
		vd.Missile1.setColor(data.Value & 0xfe)
	default:
		return true
	}

	vd.tiaHasChanged = true
	return false
}

// UpdatePlayfieldAndBackgroundColor checks TIA memory for changes to playfield color
// registers.
//
// Separate from the UpdateColor() function because some TIA revisions (or
// sometimes for some other reason eg.RGB mod) are slower when updating the
// playfield color register than the other registers.
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdatePlayfieldAndBackgroundColor(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.COLUPF:
		vd.Playfield.setColor(data.Value & 0xfe)
		vd.Ball.setColor(data.Value & 0xfe)
	case cpubus.COLUBK:
		vd.Playfield.setBackground(data.Value & 0xfe)
	default:
		return true
	}

	vd.tiaHasChanged = true
	return false
}

// UpdateSpritePixels checks TIA memory for attribute changes that *must* occur
// after a call to Pixel().
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdateSpritePixels(data chipbus.ChangedRegister) bool {
	// the barnstormer ROM demonstrate perfectly how GRP0 is affected if we
	// alter its state before a call to Pixel().  if we write do alter state
	// before Pixel(), then an unwanted artefact can be seen on scanline 61.
	switch data.Register {
	case cpubus.GRP0:
		vd.Player0.setGfxData(data.Value)
		if vd.tia.env.Prefs.Revision.Live.LateVDELGRP0.Load().(bool) {
			vd.writing.Schedule(1, 0)
			vd.writingRegister = data.Register
		} else {
			vd.Player1.setOldGfxData()
		}

	case cpubus.GRP1:
		vd.Player1.setGfxData(data.Value)
		if vd.tia.env.Prefs.Revision.Live.LateVDELGRP1.Load().(bool) {
			vd.writing.Schedule(1, 0)
			vd.writingRegister = data.Register
		} else {
			vd.Player0.setOldGfxData()
			vd.Ball.setEnableDelay()
		}

	case cpubus.ENAM0:
		vd.Missile0.setEnable(data.Value&ENAxxMask == ENAxxMask)
	case cpubus.ENAM1:
		vd.Missile1.setEnable(data.Value&ENAxxMask == ENAxxMask)
	case cpubus.ENABL:
		vd.Ball.setEnable(data.Value&ENAxxMask == ENAxxMask)
	default:
		return true
	}

	vd.tiaHasChanged = true
	return false
}

func (vd *Video) UpdateSpriteVariationsEarly(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.NUSIZ0:
		vd.Missile0.SetNUSIZ(data.Value)
	case cpubus.NUSIZ1:
		vd.Missile1.SetNUSIZ(data.Value)
	}

	// always returning true because we're only dealing with NUSIZx for missiles
	// and not players
	//
	// NOTE: setting of NUSIZx for the player sprite should probably also happen
	// at this point too. however, the SetNUSIZ() function for Player has been
	// tuned assuming that it happens later. there's a comment on the SetNUSIZ()
	// function saying that it can probably be untangled and I think that's
	// right. the key to that would be to move the call to it to this function
	return true
}

// UpdateSpriteVariations checks TIA memory for writes to registers that affect
// how sprite pixels are output. Note that CTRLPF is serviced here rather than
// in UpdatePlayfield(), because it affects the ball sprite.
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdateSpriteVariations(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.CTRLPF:
		vd.Ball.SetCTRLPF(data.Value)
		vd.Playfield.SetCTRLPF(data.Value)
	case cpubus.VDELP0:
		vd.Player0.SetVerticalDelay(data.Value&VDELPxMask == VDELPxMask)
	case cpubus.VDELP1:
		vd.Player1.SetVerticalDelay(data.Value&VDELPxMask == VDELPxMask)
	case cpubus.REFP0:
		vd.Player0.setReflection(data.Value&REFPxMask == REFPxMask)
	case cpubus.REFP1:
		vd.Player1.setReflection(data.Value&REFPxMask == REFPxMask)
	case cpubus.RESMP0:
		vd.Missile0.setResetToPlayer(data.Value&RESMPxMask == RESMPxMask)
	case cpubus.RESMP1:
		vd.Missile1.setResetToPlayer(data.Value&RESMPxMask == RESMPxMask)
	case cpubus.NUSIZ0:
		vd.Player0.setNUSIZ(data.Value)
	case cpubus.NUSIZ1:
		vd.Player1.setNUSIZ(data.Value)
	case cpubus.CXCLR:
		vd.Collisions.Clear()
	default:
		return true
	}

	vd.tiaHasChanged = true
	return false
}

// UpdateCTRLPF should be called whenever any of the individual components of
// the CTRPF are altered. For example, if Playfield.Reflected is altered, then
// this function should be called so that the CTRLPF value is set to reflect
// the alteration.
//
// This is only of use to debuggers. It's never required in normal operation of
// the emulator.
func (vd *Video) UpdateCTRLPF() {
	vd.Ball.Size &= 0x03
	ctrlpf := vd.Ball.Size << 4

	if vd.Playfield.Reflected {
		ctrlpf |= CTRLPFReflectedMask
	}
	if vd.Playfield.Scoremode {
		ctrlpf |= CTRLPFScoremodeMask
	}
	if vd.Playfield.Priority {
		ctrlpf |= CTRLPFPriorityMask
	}

	vd.Playfield.Ctrlpf = ctrlpf
	vd.Ball.Ctrlpf = ctrlpf
	vd.tiaHasChanged = true
}

// UpdateNUSIZ should be called whenever the player/missile size/copies
// information is altered. This function updates the NUSIZ value to reflect the
// changes whilst maintaining the other NUSIZ bits.
//
// This is only of use to debuggers. It's never required in normal operation of
// the emulator.
func (vd *Video) UpdateNUSIZ(num int, fromMissile bool) {
	var nusiz uint8

	if num == 0 {
		if fromMissile {
			vd.Missile0.Copies &= NUSIZxCopiesMask
			vd.Missile0.Size &= NUSIZxSizeMask
			vd.Player0.SizeAndCopies = vd.Missile0.Copies
			nusiz = vd.Missile0.Copies | vd.Missile0.Size<<4
		} else {
			vd.Player0.SizeAndCopies &= NUSIZxCopiesMask
			vd.Missile0.Copies = vd.Player0.SizeAndCopies
			nusiz = vd.Player0.SizeAndCopies | vd.Missile0.Size<<4
		}
		vd.Player0.Nusiz = nusiz
		vd.Missile0.Nusiz = nusiz
	} else {
		if fromMissile {
			vd.Missile1.Copies &= NUSIZxCopiesMask
			vd.Missile1.Size &= NUSIZxSizeMask
			vd.Player1.SizeAndCopies = vd.Missile1.Copies
			nusiz = vd.Missile1.Copies | vd.Missile1.Size<<4
		} else {
			vd.Player1.SizeAndCopies &= NUSIZxCopiesMask
			vd.Missile1.Copies = vd.Player1.SizeAndCopies
			nusiz = vd.Player1.SizeAndCopies | vd.Missile1.Size<<4
		}
		vd.Player1.Nusiz = nusiz
		vd.Missile1.Nusiz = nusiz
	}
	vd.tiaHasChanged = true
}
