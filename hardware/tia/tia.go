package tia

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/audio"
	"gopher2600/hardware/tia/delay"
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/hardware/tia/tiaclock"
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

	Clk   tiaclock.TIAClock
	hsync polycounter.Polycounter

	hmove   bool
	hmoveCt int

	// the tia keeps track of whether to send color information to the
	// television with the hblank boolean. compare to vblank which is a set and
	// unset at specific times.
	hblank bool

	// wsync records whether the cpu is to halt until hsync resets to 000000
	wsync bool

	// for clarity we think of tia video and audio as sub-systems
	Video *video.Video
	Audio *audio.Audio

	// the last signal sent to the television. many signal attributes are
	// sustained over many cycles; we use this to store that information
	sig television.SignalAttributes

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
	return tia.String()
}

// MachineInfo returns the TIA information in verbose format
func (tia TIA) MachineInfo() string {
	return tia.String()
}

// map String to MachineInfo
func (tia TIA) String() string {
	s := strings.Builder{}
	s.WriteString(polycounter.Table[tia.hsync.Count])
	s.WriteString(fmt.Sprintf(" %s %03d  %04.01f", tia.hsync, tia.videoCycles, tia.cpuCycles))

	// NOTE: TIA_HW_Notes includes playfield, pixel and control information.
	// we're choosing not to include that information here

	return s.String()
}

// NewTIA creates a TIA, to be used in a VCS emulation
func NewTIA(tv television.Television, mem memory.ChipBus) *TIA {
	tia := TIA{tv: tv, mem: mem, hblank: true}

	tia.Clk.Reset()
	tia.hsync.SetLimit(56)

	tia.Video = video.NewVideo(&tia.Clk, &tia.hsync, mem, &tia.OnFutureColorClock, &tia.OnFutureMotionClock)
	if tia.Video == nil {
		return nil
	}

	tia.Audio = audio.NewAudio()
	if tia.Audio == nil {
		return nil
	}

	return &tia
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
		tia.sig.VSync = value&0x02 == 0x02

		// TODO: do something with controller settings below
		_ = value&0x40 == 0x40
		_ = value&0x80 == 0x80
		return

	case "VBLANK":
		tia.OnFutureColorClock.Schedule(delay.TriggerVBLANK, func() {
			tia.sig.VBlank = (value&0x02 == 0x02)
		}, "setting VBLANK")
		return

	case "WSYNC":
		tia.wsync = true
		return
	case "RSYNC":
		// TODO: rsync
		return
	case "HMOVE":
		tia.Video.PrepareSpritesForHMOVE()
		tia.hmove = true
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

// StepVideoCycle moves the state of the tia forward one video cycle
// returns the state of the CPU (conceptually, we're attaching the result of
// this function to pin 3 of the 6507)
func (tia *TIA) StepVideoCycle() (bool, error) {
	// set up tv signal for this tick
	tia.sig.FrontPorch = false
	tia.sig.CBurst = false

	// update "two-phase clock generator"
	tia.Clk.Tick()

	// update debugging information
	tia.videoCycles++
	tia.cpuCycles = float64(tia.videoCycles) / 3.0

	// when phase clock reaches the correct state, tick hsync counter
	if tia.Clk.IsLatched() {
		tia.hsync.Tick()

		switch tia.hsync.Count {
		case 0:
			tia.sig.FrontPorch = true
			tia.wsync = false
			tia.videoCycles = 0
			tia.cpuCycles = 0
		case 4:
			tia.sig.HSync = true
		case 8:
			tia.sig.HSync = false
		case 13:
			tia.sig.CBurst = true
		case 16:
			// early HBLANK reset
			if tia.hmove == false {
				tia.hblank = false
			}
		case 18:
			// late HBLANK reset
			if tia.hmove == true {
				tia.hblank = false
			}
		case 56:
			tia.hblank = true
		}
	}

	// tick futures -- important that this happens after TickSprites() because
	// we want position resets in particular, have been tuned to happen after
	// sprite ticking and playfield drawing
	tia.OnFutureColorClock.Tick()
	tia.OnFutureMotionClock.Tick()

	// if !tia.hblank {
	// 	tia.Video.TickSprites()
	// }
	tia.Video.TickPlayfield()

	// HMOVE clock stuffing
	// if ct, ok := tia.Hmove.tick(); ok {
	// 	tia.Video.ResolveHorizMovement(ct)
	// }

	// resolve video pixels
	pixelColor, debugColor := tia.Video.Resolve()

	// color signal is video black in case of hblank being on
	if tia.hblank {
		tia.sig.Pixel = television.VideoBlack
	} else {
		tia.sig.Pixel = television.ColorSignal(pixelColor)
	}

	// we always send the debug color regardless of hblank
	tia.sig.AltPixel = television.ColorSignal(debugColor)

	// signal the television at the end of the video cycle
	err := tia.tv.Signal(tia.sig)

	if err != nil {
		switch err := err.(type) {
		case errors.FormattedError:
			// filter out-of-spec errors for now. this should be optional
			//if err.Errno != errors.OutOfSpec {
			return !tia.wsync, err
			//}
		default:
			return !tia.wsync, err
		}
	}

	return !tia.wsync, nil
}
