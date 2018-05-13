package tia

import (
	"fmt"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/colorclock"
	"gopher2600/hardware/tia/video"
	"gopher2600/television"
)

const vblankMask = 0x02
const vsyncMask = 0x02
const vsyncLatchTriggerMask = 0x40
const vsyncGroundedPaddleMask = 0x80

// TIA contains all the sub-components of the VCS TIA sub-system
type TIA struct {
	tv  television.Television
	mem memory.ChipBus

	colorClock *colorclock.ColorClock
	hmove      *hmove
	rsync      *rsync

	// TIA state -- controlled by the CPU
	vsync  bool
	vblank bool

	// TIA state -- set automatically by the TIA
	hblank bool
	hsync  bool
	wsync  bool

	Video *video.Video
	// TODO: audio
}

// MachineInfoTerse returns the TIA information in terse format
func (tia TIA) MachineInfoTerse() string {
	return fmt.Sprintf("%s %s %s", tia.colorClock.MachineInfoTerse(), tia.rsync.MachineInfoTerse(), tia.hmove.MachineInfoTerse())
}

// MachineInfo returns the TIA information in verbose format
func (tia TIA) MachineInfo() string {
	return fmt.Sprintf("%v\n%v\n%v", tia.colorClock, tia.rsync, tia.hmove)
}

// map String to MachineInfo
func (tia TIA) String() string {
	return tia.MachineInfo()
}

// New is the preferred method of initialisation for the TIA structure
func New(tv television.Television, mem memory.ChipBus) *TIA {
	tia := new(TIA)
	if tia == nil {
		return nil
	}

	tia.tv = tv
	tia.mem = mem

	// TODO: audio

	tia.colorClock = colorclock.New()
	if tia.colorClock == nil {
		return nil
	}

	tia.hmove = newHmove(tia.colorClock)
	if tia.hmove == nil {
		return nil
	}

	tia.rsync = newRsync(tia.colorClock)
	if tia.rsync == nil {
		return nil
	}

	tia.hblank = true

	tia.Video = video.New(tia.colorClock, &tia.hblank)
	if tia.Video == nil {
		return nil
	}

	// TODO: audio

	return tia
}

// ReadTIAMemory checks for side effects in the TIA sub-system
func (tia *TIA) ReadTIAMemory() {
	service, register, value := tia.mem.ChipRead()
	if service {
		serviced := tia.serviceTIAMemory(register, value)
		if !serviced {
			serviced = tia.Video.ServiceTIAMemory(register, value)
			if !serviced {
				// TODO: audio
				if !serviced {
					// TODO: complain that register has not been serviced
				}
			}
		}
	}
}

// serviceTIAMemory checks the TIA memory for changes to registers that are
// interesting to the top-level TIA sub-system
func (tia *TIA) serviceTIAMemory(register string, value uint8) bool {
	switch register {
	case "VSYNC":
		tia.vsync = value&vsyncMask == vsyncMask
		// TODO: do something with controller settings below
		_ = value&vsyncLatchTriggerMask == vsyncLatchTriggerMask
		_ = value&vsyncGroundedPaddleMask == vsyncGroundedPaddleMask
	case "VBLANK":
		tia.vblank = value&vblankMask == vblankMask
	case "WSYNC":
		tia.wsync = true
	case "RSYNC":
		tia.colorClock.ResetPhase()
		tia.rsync.set()
	case "HMOVE":
		tia.hmove.set()
	default:
		return false
	}
	return true
}

// StepVideoCycle moves the state of the tia forward one video cycle
// returns the state of the CPU (conceptually, we're attaching the result of
// this function to pin 3 of the 6507)
func (tia *TIA) StepVideoCycle() bool {
	frontPorch := false
	cburst := false

	if tia.colorClock.Match(16) && !tia.hmove.isActive() {
		// HBLANK off (early)
		// 011100
		tia.hblank = false
	} else if tia.colorClock.Match(18) && tia.hmove.isActive() {
		// HBLANK off (late)
		// 010111
		tia.hblank = false
	} else if tia.colorClock.Match(4) {
		// HSYNC on
		// 111100
		tia.hsync = true
	} else if tia.colorClock.Match(8) {
		// HSYNC off
		// 110111
		tia.hsync = false
	} else if tia.colorClock.Match(12) {
		// color burst
		// 001111
		cburst = true
	}

	if tia.colorClock.Tick(tia.rsync.check()) == true {
		// set up new scanline if colorClock ticks its way to its reset point or if
		// an rsync has matured (see rsync.go commentary)
		frontPorch = true
		tia.wsync = false
		tia.hblank = true
		tia.hmove.reset()
		tia.rsync.reset()
	}

	// HMOVE clock stuffing
	tia.Video.TickSpritesForHMOVE(tia.hmove.tick())

	// note that tick playfield occurs regardless of the state of hblank
	tia.Video.TickPlayfield()

	// at the end of the video cycle we want to finally 'send' information to the
	// televison. what we 'send' depends on the state of hblank.
	if !tia.hblank {
		// tick all sprites -- this is distinct from the sprite ticking we did
		// above when clock stuffing for HMOVE
		tia.Video.TickSprites()

		// send pixel color to television
		tia.tv.Signal(tia.vsync, tia.vblank, frontPorch, tia.hsync, cburst, tia.Video.PixelColor())
	} else {
		// we're in the hblank state so do not tick the sprites and send the null
		// pixel color to the television
		tia.tv.Signal(tia.vsync, tia.vblank, frontPorch, tia.hsync, cburst, video.NoColor)
	}

	return !tia.wsync
}
