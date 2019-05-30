package tia

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/audio"
	"gopher2600/hardware/tia/delay"
	"gopher2600/hardware/tia/delay/future"
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

	// there's a slight delay when changing the state of video objects. we're
	// using two future instances to emulate what happens in the 2600. the
	// first is OnFutureColorClock, which *ticks* every video cycle. we use this for
	// writing playfield bits, player bits and enable flags for missiles and
	// the ball.
	//
	// the second future instance is OnFutureMotionClock. this is for those
	// writes that only occur during the "motion clock". eg. resetting sprite
	// positions
	OnFutureColorClock  future.Group
	OnFutureMotionClock future.Group
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

	tia.Video = video.NewVideo(tia.colorClock, mem, &tia.OnFutureColorClock, &tia.OnFutureMotionClock)
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
	service, register, value := tia.mem.ChipRead()

	if !service {
		// nothing to service
		return
	}

	switch register {
	case "VSYNC":
		tia.vsync = value&0x02 == 0x02

		// TODO: do something with controller settings below
		_ = value&0x40 == 0x40
		_ = value&0x80 == 0x80
		return

	case "VBLANK":
		tia.OnFutureColorClock.Schedule(delay.TriggerVBLANK, func() {
			tia.vblank = (value&0x02 == 0x02)
		}, "setting VBLANK")
		return

	case "WSYNC":
		tia.wsync = true
		return
	case "RSYNC":
		tia.rsync.set()
		return
	case "HMOVE":
		if tia.colorClock.Count < 15 || tia.colorClock.Count >= 54 {
			// this is the regular HMOVE branch
			tia.Video.PrepareSpritesForHMOVE()
			tia.Hmove.setLatch()
		} else if tia.colorClock.Count > 39 {
			// if HMOVE is called after colorclock 39 then we "force" the HMOVE
			// instead of simply setting the HMOVE latch
			tia.Video.ForceHMOVE(-39 + tia.colorClock.Count)
		}
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

// StepVideoCycle moves the state of the tia forward one video cycle
// returns the state of the CPU (conceptually, we're attaching the result of
// this function to pin 3 of the 6507)
func (tia *TIA) StepVideoCycle() (bool, error) {
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
	} else if tia.colorClock.MatchEnd(16) {
		if !tia.Hmove.isLatched() {
			// HBLANK off (early)
			tia.hblank = false
		} else {
			// short circuit HMOVE activity but don't unlatch it
			tia.Hmove.count = -1
		}
	} else if tia.colorClock.MatchEnd(18) && tia.Hmove.isLatched() {
		// HBLANK off (late)
		tia.hblank = false
	} else if tia.colorClock.MatchEnd(54) {
		tia.Hmove.unsetLatch()
	} else {
		// motion clock is turned on/off depending on whether hmove is
		// currently active. if HMOVE is not set then motion clock is set
		// "early"; if HMOVE is set then motion clock is set "late".
		//
		// the original assumption was that motion clock would always be set
		// "early"
		//
		// this surely affects many ROMs but I first noticed the disparity when
		// viewing the Pole Position ROM
		if tia.Hmove.isLatched() {
			// set motion clock "late"
			if tia.colorClock.MatchEnd(17) {
				tia.motionClock = true
			} else if tia.colorClock.MatchEnd(1) {
				tia.motionClock = false
			}
		} else {
			// set motion clock "early"
			if tia.colorClock.MatchEnd(15) {
				tia.motionClock = true
			} else if tia.colorClock.MatchEnd(56) {
				tia.motionClock = false
			}
		}
	}

	// set up new scanline if colorClock has ticked its way to the reset point or if
	// an rsync has matured (see rsync.go commentary)
	if tia.rsync.tick() || tia.colorClock.Tick() {
		frontPorch = true
		tia.wsync = false
		tia.hblank = true
		tia.videoCycles = 0
		tia.cpuCycles = 0
		tia.colorClock.Reset()
	}

	// tick all sprites according to hblank
	if !tia.hblank {
		tia.Video.TickSprites()
	}

	// tick playfield
	tia.Video.TickPlayfield()

	// tick futures -- important that this happens after TickSprites() because
	// we want position resets in particular, have been tuned to happen after
	// sprite ticking and playfield drawing
	tia.OnFutureColorClock.Tick()
	if tia.motionClock {
		tia.OnFutureMotionClock.Tick()
	}

	// HMOVE clock stuffing
	if ct, ok := tia.Hmove.tick(); ok {
		tia.Video.ResolveHorizMovement(ct)
	}

	// decide on pixel color. we always want to do this even if HBLANK is
	// active. this is because we also set the collision registers in this
	// function and they need to be set even if there is nothing visual
	// being sent to the TV
	//
	// * Fatal Run uses collision detection on otherwise unseen pixels to turn
	// off the ball sprite being used to draw the edge of the road
	pixelColor, debugColor := tia.Video.Resolve()

	var pixel television.ColorSignal

	if tia.hblank {
		pixel = television.VideoBlack
	} else {
		pixel = television.ColorSignal(pixelColor)
	}

	// at the end of the video cycle we want to finally signal the televison
	err := tia.tv.Signal(television.SignalAttributes{
		VSync:      tia.vsync,
		VBlank:     tia.vblank,
		FrontPorch: frontPorch,
		HSync:      tia.hsync,
		CBurst:     cburst,
		Pixel:      pixel,
		AltPixel:   television.ColorSignal(debugColor)})

	if err != nil {
		switch err := err.(type) {
		case errors.FormattedError:
			// filter out-of-spec errors for now. this should be optional -
			if err.Errno != errors.OutOfSpec {
				return !tia.wsync, err
			}
		default:
			return !tia.wsync, err
		}
	}

	return !tia.wsync, nil
}
