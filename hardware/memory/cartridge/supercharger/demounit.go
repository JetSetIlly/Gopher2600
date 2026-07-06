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

package supercharger

import (
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/notifications"
)

// The Supercharger demo unit is a device that plugs into the supercharger and also the right player
// port of the 2600. Emulation of the device is therefore split over two mechanisms. This DemoUnit
// type handles the initial loading of the bootloader via the normal Supercharger method. It also
// handle the plugging in on the controller.
//
// https://forums.atariage.com/topic/390261-starpath-demonstration-unit-rom-dump/
type DemoUnit struct {
	env        *environment.Environment
	schweber   []uint8
	bootloader []uint8
	jmpAddrLo  uint8
	jmpAddrHi  uint8
	configByte uint8
	controller *demoUnit_controller
}

const SchweberHash = "0a98bc3d53a0965de87fc77377b5a0db"

func newDemoUnit(env *environment.Environment) (*DemoUnit, error) {
	if env.Loader.HashMD5 != SchweberHash {
		return nil, fmt.Errorf("demo unit: exepected 'Schweber OQ' ROM dump")
	}

	data, err := io.ReadAll(env.Loader)
	if err != nil {
		return nil, fmt.Errorf("demo unit: %w", err)
	}

	dem := &DemoUnit{
		env:        env,
		schweber:   data[:],
		bootloader: data[0x300:0x400],
		jmpAddrLo:  data[0x301],
		jmpAddrHi:  data[0x302],
		configByte: data[0x303],
	}

	return dem, nil
}

// snapshot implements the tape interface.
func (dem *DemoUnit) snapshot() tape {
	n := *dem
	return &n
}

// plumb implements the tape interface.
func (dem *DemoUnit) plumb(env *environment.Environment) {
	dem.env = env
}

// load implements the tape interface.
func (dem *DemoUnit) load() (uint8, error) {
	// if controller has already been attached then just return a zero byte
	if dem.controller != nil {
		return 0x00, nil
	}

	err := dem.env.Notifications.Notify(notifications.NotifySuperchargerFastLoad)
	if err != nil {
		return 0x00, fmt.Errorf("fastload: %w", err)
	}
	return 0x00, nil
}

// step implements the tape interface.
func (dem *DemoUnit) step() {
}

// load implements the tape interface.
func (tap *DemoUnit) end() {
}

// bootstrap implements the tape interface
func (dem *DemoUnit) bootstrap(state *state, mc *cpu.CPU, ram *vcs.RAM, riot *riot.RIOT, tia *tia.TIA) error {
	// copy bootloader to correct location, such that the start instruction is at $ffc0
	clear(state.ram[0])
	clear(state.ram[1])
	clear(state.ram[2])
	copy(state.ram[1][0x6f5:], dem.bootloader)

	// same RAM initialisation as fastload
	for i := uint16(0x0082); i <= 0x009d; i++ {
		_ = ram.Poke(i, 0x00)
	}
	_ = ram.Poke(0x80, dem.configByte)

	// same boot intialisation sequence as fastload
	_ = ram.Poke(0xfa, 0xcd)
	_ = ram.Poke(0xfb, 0xf8)
	_ = ram.Poke(0xfc, 0xff)
	_ = ram.Poke(0xfd, 0x4c)
	_ = ram.Poke(jmpAddrLo, dem.jmpAddrLo)
	_ = ram.Poke(jmpAddrHi, dem.jmpAddrHi)

	// same quickBootstrap choice as fastload
	if quickBootstrap {
		mc.PC.Load(uint16(dem.jmpAddrLo) | uint16(dem.jmpAddrHi)<<8)
		state.registers.setConfigByte(dem.configByte)
	} else {
		err := mc.LoadPC(0x00fa)
		if err != nil {
			return fmt.Errorf("demo unit: %w", err)
		}
		state.registers.Value = dem.configByte
		state.registers.Delay = 0
	}

	// same state changes as what we discovered for fastload
	riot.Timer.PokeField("divider", timer.TIM64T)
	riot.Timer.PokeField("ticksRemaining", 0x1f)
	riot.Timer.PokeField("intim", uint8(0x0a))
	riot.Timer.PokeField("pa7", false)
	tia.Video.Player0.SetVerticalDelay(false)
	tia.Video.Player1.SetVerticalDelay(false)
	tia.Video.Player0.SetNUSIZ(0)
	tia.Video.Player1.SetNUSIZ(0)
	tia.Video.Ball.Hmove = 8

	// the demo unit is also plugged into the console's joystick port
	err := riot.Ports.Plug(plugging.PortRight,
		func(env *environment.Environment, id plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
			p := newDemoUnitController(env, id, bus, dem.schweber)
			if p == nil {
				return nil
			}
			dem.controller = p.(*demoUnit_controller)
			return p
		},
	)
	if err != nil {
		return fmt.Errorf("demo unit: %w", err)
	}

	return nil
}

func (dem *DemoUnit) romdump(w io.Writer) error {
	return fmt.Errorf("demo unit: romdump: unsupported")
}

func (dem *DemoUnit) jmpAddr() uint16 {
	return uint16(dem.jmpAddrLo) | uint16(dem.jmpAddrHi)<<8
}
