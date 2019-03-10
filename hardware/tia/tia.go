package tia

import (
	"fmt"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/audio"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/hardware/tia/video"
	"gopher2600/television"
	"strings"
)

const vblankMask = 0x02
const vsyncMask = 0x02
const vsyncLatchTriggerMask = 0x40
const vsyncGroundedPaddleMask = 0x80

// TIA contains all the sub-components of the VCS TIA sub-system
type TIA struct {
	tv  television.Television
	mem memory.ChipBus

	// number of cycles since the last WSYNC
	cpuCycles   int
	videoCycles int

	colorClock *polycounter.Polycounter

	// motion clock is an out-of-phase colorClock, running 2 cycles ahead of
	// the main color clock (according to the document, "Atari 2600 TIA
	// Hardware Notes" by Andrew Towers) - currently used to indicate when
	// calling tickFutures()
	motionClock bool

	Hmove *hmove
	rsync *rsync

	// TIA state -- controlled by the CPU
	vsync  bool
	vblank bool

	// TIA state -- set automatically by the TIA
	hblank bool
	hsync  bool
	wsync  bool

	Video *video.Video
	Audio *audio.Audio
}

// MachineInfoTerse returns the TIA information in terse format
func (tia TIA) MachineInfoTerse() string {
	return fmt.Sprintf("%s %s %s", tia.colorClock.MachineInfoTerse(), tia.rsync.MachineInfoTerse(), tia.Hmove.MachineInfoTerse())
}

// MachineInfo returns the TIA information in verbose format
func (tia TIA) MachineInfo() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("TIA:\n   colour clock: %v\n   %v\n   %v\n", tia.colorClock, tia.rsync, tia.Hmove))
	s.WriteString(fmt.Sprintf("   Cycles since WSYNC:\n     CPU=%d\n     Video=%d", tia.cpuCycles, tia.videoCycles))
	return s.String()
}

// map String to MachineInfo
func (tia TIA) String() string {
	return tia.MachineInfo()
}

// NewTIA creates a TIA, to be used in a VCS emulation
func NewTIA(tv television.Television, mem memory.ChipBus) *TIA {
	tia := new(TIA)
	tia.tv = tv
	tia.mem = mem

	tia.colorClock = polycounter.New6Bit()
	tia.colorClock.SetResetPoint(56) // "010100"

	tia.Hmove = newHmove(tia.colorClock)
	if tia.Hmove == nil {
		return nil
	}

	tia.rsync = newRsync(tia.colorClock)
	if tia.rsync == nil {
		return nil
	}

	tia.hblank = true

	tia.Video = video.NewVideo(tia.colorClock, mem)
	if tia.Video == nil {
		return nil
	}

	tia.Audio = audio.NewAudio()
	if tia.Audio == nil {
		return nil
	}

	return tia
}

// ReadTIAMemory checks for side effects in the TIA sub-system
func (tia *TIA) ReadTIAMemory() {
	valueRead, register, value := tia.mem.ChipRead()

	if !valueRead {
		// nothing to service
		return
	}

	switch register {
	case "VSYNC":
		tia.vsync = value&vsyncMask == vsyncMask
		// TODO: do something with controller settings below
		_ = value&vsyncLatchTriggerMask == vsyncLatchTriggerMask
		_ = value&vsyncGroundedPaddleMask == vsyncGroundedPaddleMask
		return
	case "VBLANK":
		tia.vblank = value&vblankMask == vblankMask
		return
	case "WSYNC":
		tia.wsync = true
		return
	case "RSYNC":
		tia.rsync.set()
		return
	case "HMOVE":
		if tia.colorClock.Count < 15 {
			tia.Video.PrepareSpritesForHMOVE()
			tia.Hmove.set()
		} else if tia.colorClock.Count > 39 && tia.colorClock.Count < 55 {
			tia.Video.ForceHMOVE(-39 + tia.colorClock.Count)
		} else if tia.colorClock.Count >= 54 {
			tia.Video.PrepareSpritesForHMOVE()
			tia.Hmove.set()
		}
		return
	}

	if tia.Video.ReadVideoMemory(register, value) {
		return
	}

	if tia.Audio.ReadAudioMemory(register, value) {
		return
	}

	panic(fmt.Errorf("unserviced register (%s=%v)", register, value))
}

// StepVideoCycle moves the state of the tia forward one video cycle
// returns the state of the CPU (conceptually, we're attaching the result of
// this function to pin 3 of the 6507)
func (tia *TIA) StepVideoCycle() bool {
	tia.videoCycles++
	if tia.videoCycles%3 == 0 {
		tia.cpuCycles++
	}

	frontPorch := false
	cburst := false

	// color clock
	if tia.colorClock.MatchEnd(4) {
		tia.hsync = true
	} else if tia.colorClock.MatchEnd(8) {
		tia.hsync = false
	} else if tia.colorClock.MatchEnd(12) {
		cburst = true
	} else if tia.colorClock.MatchEnd(15) {
		tia.motionClock = true
	} else if tia.colorClock.MatchEnd(16) {
		if !tia.Hmove.isset() {
			// HBLANK off (early)
			tia.hblank = false
		}
		tia.Hmove.count = -1
	} else if tia.colorClock.MatchEnd(18) && tia.Hmove.isset() {
		// HBLANK off (late)
		tia.hblank = false
	} else if tia.colorClock.MatchEnd(54) {
		tia.Hmove.unset()
	} else if tia.colorClock.MatchEnd(56) {
		tia.motionClock = false
	}

	// set up new scanline if colorClock has ticked its way to the reset point or if
	// an rsync has matured (see rsync.go commentary)
	if tia.rsync.tick() {
		frontPorch = true
		tia.wsync = false
		tia.hblank = true
		tia.Hmove.unset()
		tia.colorClock.Reset()
	} else if tia.colorClock.Tick() {
		frontPorch = true
		tia.wsync = false
		tia.hblank = true
		tia.videoCycles = 0
		tia.cpuCycles = 0
		// not sure if we need to reset rsync
	}

	// HMOVE clock stuffing
	if ct, ok := tia.Hmove.tick(); ok {
		tia.Video.ResolveHorizMovement(ct)
	}

	// tick all sprites according to hblank
	if !tia.hblank {
		tia.Video.TickSprites()
	}

	// tick playfield and scheduled writes
	// -- important that this happens after TickSprites because we want
	// position resets to happen *after* sprite ticking; in particular, when
	// the draw signal has been resolved
	tia.Video.TickPlayfield()
	tia.Video.TickFutures(tia.motionClock)

	// decide on pixel color
	var pixelColor, debugColor uint8
	if !tia.hblank {
		pixelColor, debugColor = tia.Video.Pixel()
	}

	// at the end of the video cycle we want to finally signal the televison
	err := tia.tv.Signal(television.SignalAttributes{
		VSync:      tia.vsync,
		VBlank:     tia.vblank,
		FrontPorch: frontPorch,
		HSync:      tia.hsync,
		CBurst:     cburst,
		Pixel:      television.ColorSignal(pixelColor),
		AltPixel:   television.ColorSignal(debugColor)})
	if err != nil {
		panic(err)
	}

	return !tia.wsync
}
