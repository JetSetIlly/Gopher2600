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

package hardware

import (
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/controllers"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/panel"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/logger"
)

// The number of times the TIA updates every CPU cycle.
const ColorClocksPerCPUCycle = 3

// VCS struct is the main container for the emulated components of the VCS.
type VCS struct {
	Instance *instance.Instance

	// the television is not "part" of the VCS console but it's part of the VCS system
	TV *television.Television

	// references to the different components of the VCS. do not take copies of
	// these pointer values because the rewind feature will change them.
	CPU  *cpu.CPU
	Mem  *memory.Memory
	RIOT *riot.RIOT
	TIA  *tia.TIA

	// The Clock defines the basic speed at which the the machine is runningt. This governs
	// the speed of the CPU, the RIOT and attached peripherals. The TIA runs at
	// exactly three times this speed.
	//
	// The different clock speeds are due to the nature of the different TV
	// specifications. Put simply, a PAL machine must run slightly slower in
	// order to be able to send a correct PAL signal to the television.
	//
	// Unlike the real hardware however, it is not the console that governs the
	// clock speed but the television. A ROM will send a signal to the
	// television, the timings of which will be used by the tv implementation
	// to decide what type of TV signal (PAL or NTSC) is being sent. When the
	// television detects a change in the TV signal it will notify the emulated
	// console, allowing it to note the new implied clock speed.
	Clock float32
}

// NewVCS creates a new VCS and everything associated with the hardware. It is
// used for all aspects of emulation: debugging sessions, and regular play.
//
// The Instance.Context field should be updated except in the case of the
// "main" emulation.
func NewVCS(tv *television.Television) (*VCS, error) {
	// set up instance
	instance, err := instance.NewInstance(tv)
	if err != nil {
		return nil, err
	}

	// set up hardware
	vcs := &VCS{
		Instance: instance,
		TV:       tv,
		Clock:    ntscClock,
	}

	vcs.Mem = memory.NewMemory(vcs.Instance)
	vcs.CPU = cpu.NewCPU(vcs.Instance, vcs.Mem)
	vcs.RIOT = riot.NewRIOT(vcs.Instance, vcs.Mem.RIOT, vcs.Mem.TIA)

	vcs.TIA, err = tia.NewTIA(vcs.Instance, vcs.TV, vcs.Mem.TIA, vcs.RIOT.Ports, vcs.CPU)
	if err != nil {
		return nil, err
	}

	err = vcs.RIOT.Ports.Plug(plugging.PortLeftPlayer, controllers.NewAuto)
	if err != nil {
		return nil, err
	}

	err = vcs.RIOT.Ports.Plug(plugging.PortRightPlayer, controllers.NewAuto)
	if err != nil {
		return nil, err
	}

	err = vcs.RIOT.Ports.Plug(plugging.PortPanel, panel.NewPanel)
	if err != nil {
		return nil, err
	}

	vcs.TV.AttachVCS(vcs)

	return vcs, nil
}

// End cleans up any resources that may be dangling.
func (vcs *VCS) End() {
	vcs.TV.End()
}

// AttachCartridge to this VCS. While this function can be called directly it
// is advised that the setup package be used in most circumstances.
func (vcs *VCS) AttachCartridge(cartload cartridgeloader.Loader) error {
	err := vcs.TV.SetSpecConditional(cartload.Spec)
	if err != nil {
		return err
	}

	if cartload.Filename == "" {
		vcs.Mem.Cart.Eject()
	} else {
		err := vcs.Mem.Cart.Attach(cartload)
		if err != nil {
			return err
		}
	}

	// resetting after cartridge attachment because the cartridge needs a reset
	// too. we could "correct" this my mandating every cartridge mapper goes
	// through the reset procedure on initialisation, but this feels safer
	// somehow.

	err = vcs.Reset()
	if err != nil {
		return err
	}

	return nil
}

// Reset emulates the reset switch on the console panel.
func (vcs *VCS) Reset() error {
	err := vcs.TV.Reset(false)
	if err != nil {
		return err
	}

	// easiest way of resetting the TIA is to just create new one
	//
	// 27/10/21 - we do want to save the audio though in order to keep any
	// attached trackers
	//
	// TODO: proper Reset() function for the TIA
	audio := vcs.TIA.Audio
	vcs.TIA, err = tia.NewTIA(vcs.Instance, vcs.TV, vcs.Mem.TIA, vcs.RIOT.Ports, vcs.CPU)
	if err != nil {
		return err
	}
	vcs.TIA.Audio = audio

	// other areas of the VCS are simply reset because the emulation may have
	// altered the part of the state that we do *not* want to reset. notably,
	// memory may have a cartridge attached - we wouldn't want to lose that.

	vcs.Mem.Reset()
	vcs.CPU.Reset()
	vcs.RIOT.Timer.Reset()

	// reset of ports must happen after reset of memory because ports will
	// update memory to the current state of the peripherals
	vcs.RIOT.Ports.ResetPeripherals()

	// reset PC using reset address in cartridge memory
	err = vcs.CPU.LoadPCIndirect(addresses.Reset)
	if err != nil {
		if !curated.Is(err, cartridge.Ejected) {
			return err
		}
	}

	// reset cart after loaded PC value. this seems unnecessary but some
	// cartridge types may switch banks on LoadPCIndirect() - those that switch
	// on Listen() - this is an artefact of the emulation method so we need to make
	// sure it's initialised correctly.
	vcs.Mem.Cart.Reset()

	return nil
}

const (
	ntscClock = 1.193182
	palClock  = 1.182298
)

// SetClockSpeed is an implemtation of the television.VCSReturnChannel interface.
func (vcs *VCS) SetClockSpeed(tvSpec string) error {
	switch tvSpec {
	case "NTSC":
		if vcs.Clock != ntscClock {
			vcs.Clock = ntscClock
			logger.Log("vcs", "switching to NTSC clock")
		}
	case "PAL":
		if vcs.Clock != palClock {
			logger.Log("vcs", "switching to PAL clock")
			vcs.Clock = palClock
		}
	}
	return curated.Errorf("vcs: cannot set clock speed for unknown tv specification (%s)", tvSpec)
}

// DetatchEmulationExtras removes all possible monitors, recorders, etc. from
// the emulation.  Currently this mean: the TIA audio tracker, the RIOT event
// recorders and playback, and RIOT plug monitor.
func (vcs *VCS) DetatchEmulationExtras() {
	vcs.TIA.Audio.SetTracker(nil)
	vcs.RIOT.Ports.AttachEventRecorder(nil)
	vcs.RIOT.Ports.AttachPlayback(nil)
	vcs.RIOT.Ports.AttachPlugMonitor(nil)
}

// HandleInputEvent forwards an input event to VCS Ports.
//
// It's important that this function be used in preference to calling the
// HandleInputEvent() function in the RIOT.Ports directly. For rewind reasons
// it is likely that any direct reference to RIOT.Ports will grow stale.
func (vcs *VCS) HandleInputEvent(inp ports.InputEvent) (bool, error) {
	return vcs.RIOT.Ports.HandleInputEvent(inp)
}

// PeripheralID forwards a request of the PeripheralID of the PortID to VCS
// Ports.
//
// See important comment in HandleInputEvent() above.
func (vcs *VCS) PeripheralID(id plugging.PortID) plugging.PeripheralID {
	return vcs.RIOT.Ports.PeripheralID(id)
}

// QueueEvent forwards an input event to VCS Ports.
func (vcs *VCS) QueueEvent(inp ports.InputEvent) error {
	return vcs.RIOT.Ports.QueueEvent(inp)
}
