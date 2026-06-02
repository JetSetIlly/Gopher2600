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
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/notifications"
)

type Schweber struct {
	env        *environment.Environment
	bootloader []uint8
	jmpAddrLo  uint8
	jmpAddrHi  uint8
	configByte uint8
}

const SchweberHash = "0a98bc3d53a0965de87fc77377b5a0db"

func newSchweber(env *environment.Environment) (*Schweber, error) {
	if env.Loader.HashMD5 != SchweberHash {
		return nil, fmt.Errorf("demo unit: exepected 'Schweber OQ' ROM dump")
	}

	data, err := io.ReadAll(env.Loader)
	if err != nil {
		return nil, fmt.Errorf("demo unit: %w", err)
	}

	fl := &Schweber{
		env:        env,
		bootloader: data[0x300:0x400],
		jmpAddrLo:  data[0x301],
		jmpAddrHi:  data[0x302],
		configByte: data[0x303],
	}

	return fl, nil
}

// snapshot implements the tape interface.
func (fl *Schweber) snapshot() tape {
	n := *fl
	return &n
}

// plumb implements the tape interface.
func (fl *Schweber) plumb(env *environment.Environment) {
	fl.env = env
}

// load implements the tape interface.
func (fl *Schweber) load() (uint8, error) {
	err := fl.env.Notifications.Notify(notifications.NotifySuperchargerFastLoad)
	if err != nil {
		return 0x00, fmt.Errorf("fastload: %w", err)
	}
	return 0x00, nil
}

// step implements the tape interface.
func (fl *Schweber) step() {
}

// load implements the tape interface.
func (tap *Schweber) end() {
}

// bootstrap implements the tape interface
func (fl *Schweber) bootstrap(state *state, mc *cpu.CPU, ram *vcs.RAM, riot *riot.RIOT, tia *tia.TIA) error {
	// copy bootloader to correct location, such that the start instruction is at $ffc0
	clear(state.ram[0])
	clear(state.ram[1])
	clear(state.ram[2])
	copy(state.ram[1][0x6f5:], fl.bootloader)

	// same RAM initialisation as fastload
	for i := uint16(0x0082); i <= 0x009d; i++ {
		_ = ram.Poke(i, 0x00)
	}
	_ = ram.Poke(0x80, fl.configByte)

	// same boot intialisation sequence as fastload
	_ = ram.Poke(0xfa, 0xcd)
	_ = ram.Poke(0xfb, 0xf8)
	_ = ram.Poke(0xfc, 0xff)
	_ = ram.Poke(0xfd, 0x4c)
	_ = ram.Poke(jmpAddrLo, fl.jmpAddrLo)
	_ = ram.Poke(jmpAddrHi, fl.jmpAddrHi)

	// same quickBootstrap choice as fastload
	if quickBootstrap {
		mc.PC.Load(uint16(fl.jmpAddrLo) | uint16(fl.jmpAddrHi)<<8)
		state.registers.setConfigByte(fl.configByte)
	} else {
		err := mc.LoadPC(0x00fa)
		if err != nil {
			return fmt.Errorf("demo unit: %w", err)
		}
		state.registers.Value = fl.configByte
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

	return nil
}

func (fl *Schweber) romdump(w io.Writer) error {
	return fmt.Errorf("demo unit: romdump: unsupported")
}

func (fl *Schweber) jmpAddr() uint16 {
	return uint16(fl.jmpAddrLo) | uint16(fl.jmpAddrHi)<<8
}
