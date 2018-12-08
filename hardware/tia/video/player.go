package video

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/hardware/tia/video/future"
	"math/bits"
	"strings"
)

type playerSprite struct {
	*sprite

	// additional sprite information
	color         uint8
	size          uint8
	verticalDelay bool
	gfxData       uint8
	gfxDataPrev   uint8
	gfxDataDelay  *uint8
	gfxDataOther  *uint8
	reflected     bool

	// the list of color clock states when missile drawing is triggered
	triggerList []int

	// if any of the sprite's draw positions are reached but a reset position
	// signal has been scheduled, then we need to delay the start of the
	// sprite's graphics scan. the drawing actually commences when the reset
	// actually takes place (concept shared with missile sprite)
	deferDrawStart bool

	// this is a wierd one. if a reset has occured and the missile is about to
	// start drawing on the next tick, then resetTriggeredOnDraw is set to true
	// the deferDrawSig is then set to the opposite value when the draw is
	// supposed to start (concept shared with missile sprite)
	resetTriggeredOnDraw bool

	// unlike missile and ball sprites, the player sprite does not always allow
	// its graphics scan counter to tick. for double and quadruple width player
	// sprites, it ticks only evey other and every fourth color clock
	// respectively. the graphicsScanFilter field is ticked every time the
	// sprite is ticked but the graphics scan counter is ticked only when
	// (depending on size) mod 1, mod 2 or mod 4 equals 0
	graphicsScanFilter int
}

func newPlayerSprite(label string, colorClock *polycounter.Polycounter) *playerSprite {
	ps := new(playerSprite)
	ps.sprite = newSprite(label, colorClock)
	return ps
}

// because of the delay in starting pixel output with player sprites we are
// adding one to our reported pixel start position (with additional pixels
// for the larger player sizes)
func (ps playerSprite) visualPixel() int {
	visPix := ps.horizPos + 1
	if ps.size == 0x05 || ps.size == 0x07 {
		visPix++
	}
	return visPix
}

// MachineInfo returns the player sprite information in terse format
func (ps playerSprite) MachineInfoTerse() string {
	return fmt.Sprintf("%s (vis pix=%d)", ps.sprite.MachineInfoTerse(), ps.visualPixel())
}

// MachineInfo returns the player sprite information in verbose format
func (ps playerSprite) MachineInfo() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("   visual pixel: %d\n", ps.visualPixel()))
	s.WriteString(fmt.Sprintf("   color: %d\n", ps.color))
	s.WriteString(fmt.Sprintf("   size: %d\n", ps.size))
	if ps.verticalDelay {
		s.WriteString("   vert delay: yes\n")
		s.WriteString(fmt.Sprintf("   gfx: %08b\n", *ps.gfxDataDelay))
		s.WriteString(fmt.Sprintf("   other gfx: %08b\n", ps.gfxData))
	} else {
		s.WriteString("   vert delay: no\n")
		s.WriteString(fmt.Sprintf("   gfx: %08b\n", ps.gfxData))
		s.WriteString(fmt.Sprintf("   other gfx: %08b\n", *ps.gfxDataDelay))
	}
	if ps.reflected {
		s.WriteString("   reflected: yes")
	} else {
		s.WriteString("   reflected: no")
	}

	return fmt.Sprintf("%s%s", ps.sprite.MachineInfo(), s.String())
}

// tick moves the counters along for the player sprite
func (ps *playerSprite) tick() {
	// position
	if ps.tickPosition(ps.triggerList) {
		if ps.resetting && !ps.resetTriggeredOnDraw {
			ps.deferDrawStart = true
		} else {
			ps.startDrawing()
		}

		if ps.size == 0x05 {
			ps.graphicsScanFilter = 1
		} else if ps.size == 0x07 {
			ps.graphicsScanFilter = 3
		}

	} else {
		// if player.position.tick() has not caused the position counter to
		// cycle then progress draw signal according to color clock phase and
		// nusiz_player_width. for nusiz_player_width and 0b101 and 0b111,
		// pixels are smeared over additional cycles in order to create the
		// double and quadruple sized sprites
		if ps.size == 0x05 {
			if ps.graphicsScanFilter%2 == 0 {
				ps.tickGraphicsScan()
			}
		} else if ps.size == 0x07 {
			if ps.graphicsScanFilter%4 == 0 {
				ps.tickGraphicsScan()
			}
		} else {
			ps.tickGraphicsScan()
		}

		if !ps.deferDrawStart {
			ps.graphicsScanFilter++
		}
	}
}

// pixel returns the color of the player at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (ps *playerSprite) pixel() (bool, uint8) {
	// vertical delay
	gfxData := ps.gfxData
	if ps.verticalDelay {
		gfxData = *ps.gfxDataDelay
	}

	// reflection
	if ps.reflected {
		gfxData = bits.Reverse8(gfxData)
	}

	// player sprites are unusual in that the first tick of the draw signal is
	// discounted
	if ps.isDrawing() && ps.graphicsScanCounter > 0 {
		if gfxData>>(uint8(ps.graphicsScanMax)-uint8(ps.graphicsScanCounter))&0x01 == 0x01 {
			return true, ps.color
		}
	}

	return false, 0
}

func (ps *playerSprite) scheduleReset(onFutureWrite *future.Group) {
	ps.resetting = true
	ps.resetTriggeredOnDraw = ps.position.CycleOnNextTick()

	onFutureWrite.Schedule(delayResetPlayer, func() {
		ps.resetting = false
		ps.resetTriggeredOnDraw = false
		ps.resetPosition()
		if ps.deferDrawStart {
			ps.startDrawing()
			ps.deferDrawStart = false
		}
	}, fmt.Sprintf("%s resetting", ps.label))
}

func (ps *playerSprite) scheduleWrite(data uint8, onFutureWrite *future.Group) {
	onFutureWrite.Schedule(delayWritePlayer, func() {
		ps.gfxDataPrev = *ps.gfxDataOther
		ps.gfxData = data
	}, fmt.Sprintf("%s writing data", ps.label))
}

func (ps *playerSprite) scheduleVerticalDelay(delay bool, onFutureWrite *future.Group) {
	label := "enabling vertical delay"
	if !delay {
		label = "disabling vertical delay"
	}

	onFutureWrite.Schedule(delayVDELP, func() {
		ps.verticalDelay = delay
	}, fmt.Sprintf("%s %s", ps.label, label))
}
