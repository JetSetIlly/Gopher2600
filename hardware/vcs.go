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
	"errors"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/clocks"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/input"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/peripherals"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/controllers"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/panel"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
)

// The number of times the TIA updates every CPU cycle.
const ColorClocksPerCPUCycle = 3

// VCS struct is the main container for the emulated components of the VCS.
type VCS struct {
	Env *environment.Environment

	// the television is not "part" of the VCS console but it's part of the VCS system
	TV *television.Television

	// references to the different sub-systems of the VCS. these syb-systems can be
	// copied with the Snapshot() function creating an instance of the State type
	CPU  *cpu.CPU
	Mem  *memory.Memory
	RIOT *riot.RIOT
	TIA  *tia.TIA

	// the input sub-system. this is not part of the Snapshot() process
	Input *input.Input

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
// The Label argument indicates which environment the emulation will be
// happening in. This affects how log entries are handled, amonst other things
//
// The Television argument should not be nil. The Notify and Preferences
// argument may be nil if required.
func NewVCS(label environment.Label, tv *television.Television, notify notifications.Notify, prefs *preferences.Preferences) (*VCS, error) {
	// set up environment
	env, err := environment.NewEnvironment(label, tv, notify, prefs)
	if err != nil {
		return nil, err
	}

	// set up hardware
	vcs := &VCS{
		Env:   env,
		TV:    tv,
		Clock: clocks.NTSC,
	}

	vcs.Mem = memory.NewMemory(vcs.Env)
	vcs.CPU = cpu.NewCPU(vcs.Mem)
	vcs.RIOT = riot.NewRIOT(vcs.Env, vcs.Mem.RIOT, vcs.Mem.TIA)

	vcs.Input = input.NewInput(vcs.TV, vcs.RIOT.Ports)

	vcs.TIA, err = tia.NewTIA(vcs.Env, vcs.TV, vcs.Mem.TIA, vcs.RIOT.Ports, vcs.CPU)
	if err != nil {
		return nil, err
	}

	err = vcs.RIOT.Ports.Plug(plugging.PortLeft, controllers.NewStick)
	if err != nil {
		return nil, err
	}

	err = vcs.RIOT.Ports.Plug(plugging.PortRight, controllers.NewStick)
	if err != nil {
		return nil, err
	}

	err = vcs.RIOT.Ports.Plug(plugging.PortPanel, panel.NewPanel)
	if err != nil {
		return nil, err
	}

	vcs.TV.AttachVCS(env, vcs)

	return vcs, nil
}

// End cleans up any resources that may be dangling.
func (vcs *VCS) End() {
	vcs.TV.End()
	vcs.RIOT.Ports.End()
}

// AttachCartridge to this VCS. While this function can be called directly it
// is advised that the setup package be used in most circumstances.
//
// The emulated VCS is *not* reset after AttachCartridge()
func (vcs *VCS) AttachCartridge(cartload cartridgeloader.Loader) (rerr error) {
	vcs.Env.Loader = cartload

	if vcs.Env.Loader.Filename == "" {
		vcs.Mem.Cart.Eject()
	} else {
		err := vcs.Mem.Cart.Attach(vcs.Env.Loader)
		if err != nil {
			return err
		}

		// fingerprint new peripherals. peripherals are not changed if option is not set
		err = vcs.FingerprintPeripheral(plugging.PortLeft)
		if err != nil {
			return err
		}
		err = vcs.FingerprintPeripheral(plugging.PortRight)
		if err != nil {
			return err
		}

	}

	// reset machine
	err := vcs.Reset()
	if err != nil {
		return err
	}

	return nil
}

// FingerprintPeripheral inserts the peripheral that is thought to be best
// suited for the current inserted cartridge.
func (vcs *VCS) FingerprintPeripheral(id plugging.PortID) error {
	return vcs.RIOT.Ports.Plug(id, peripherals.Fingerprint(id, vcs.Env.Loader))
}

// Reset emulates the reset switch on the console panel.
func (vcs *VCS) Reset() error {
	err := vcs.TV.Reset(false)
	if err != nil {
		return err
	}

	// attaching the TV spec is done after a call to tv.Reset(). this happens
	// even if the cartridge has already been attached because we may reset the
	// console after ROM selection
	if vcs.TV.IsAutoSpec() {
		err = vcs.TV.SetSpec(vcs.Env.Loader.ReqSpec)
		if err != nil {
			return err
		}
	}

	// easiest way of resetting the TIA is to just create new one
	//
	// 27/10/21 - we do want to save the audio though in order to keep any
	// attached trackers
	//
	// TODO: proper Reset() function for the TIA
	audio := vcs.TIA.Audio
	vcs.TIA, err = tia.NewTIA(vcs.Env, vcs.TV, vcs.Mem.TIA, vcs.RIOT.Ports, vcs.CPU)
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

	// randomise CPU registers
	if vcs.Env.Prefs.RandomState.Get().(bool) {
		vcs.CPU.PC.Load(uint16(vcs.Env.Random.NoRewind(0xffff)))
		vcs.CPU.A.Load(uint8(vcs.Env.Random.NoRewind(0xff)))
		vcs.CPU.X.Load(uint8(vcs.Env.Random.NoRewind(0xff)))
		vcs.CPU.Y.Load(uint8(vcs.Env.Random.NoRewind(0xff)))
		vcs.CPU.SP.Load(uint8(vcs.Env.Random.NoRewind(0xff)))
		vcs.CPU.Status.Load(uint8(vcs.Env.Random.NoRewind(0xff)))
	}

	// reset PC using reset address in cartridge memory
	err = vcs.CPU.LoadPCIndirect(cpu.Reset)
	if err != nil {
		if !errors.Is(err, cartridge.Ejected) {
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

// SetClockSpeed is an implemtation of the television.VCSReturnChannel interface.
func (vcs *VCS) SetClockSpeed(spec specification.Spec) {
	switch spec.ID {
	case specification.SpecNTSC.ID:
		if vcs.Clock != clocks.NTSC {
			vcs.Clock = clocks.NTSC
			logger.Log(vcs.Env, "vcs", "switching to NTSC clock")
		}
	case specification.SpecPAL.ID:
		if vcs.Clock != clocks.PAL {
			vcs.Clock = clocks.PAL
			logger.Log(vcs.Env, "vcs", "switching to PAL clock")
		}
	case specification.SpecPAL_M.ID:
		if vcs.Clock != clocks.PAL_M {
			vcs.Clock = clocks.PAL_M
			logger.Log(vcs.Env, "vcs", "switching to PAL-M clock")
		}
	case specification.SpecSECAM.ID:
		if vcs.Clock != clocks.SECAM {
			vcs.Clock = clocks.SECAM
			logger.Log(vcs.Env, "vcs", "switching to SECAM clock")
		}
	default:
		logger.Logf(vcs.Env, "vcs", "cannot set clock for unknown TV specification (%s)", spec.ID)
	}
}

// DetatchEmulationExtras removes all possible monitors, recorders, etc. from
// the emulation.  Currently this mean: the TIA audio tracker, the RIOT event
// recorders and playback, and RIOT plug monitor.
func (vcs *VCS) DetatchEmulationExtras() {
	vcs.TIA.Audio.SetTracker(nil)
	vcs.Input.ClearRecorders()
	vcs.Input.AttachPlayback(nil)
	vcs.RIOT.Ports.AttachPlugMonitor(nil)
}
