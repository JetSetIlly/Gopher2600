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

package cartridge

import (
	"fmt"
	"io"
	"os"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper/banking"
	"github.com/jetsetilly/gopher2600/logger"
)

const DevCardID = "devcard"

type DevCard struct {
	env *environment.Environment
	mem []uint8
}

func newDevCard(env *environment.Environment) (mapper.CartMapper, error) {
	cart := &DevCard{
		env: env,
	}

	data, err := io.ReadAll(env.Loader)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", DevCardID, err)
	}

	cart.mem = make([]uint8, 0x6000)
	copy(cart.mem, data)

	return cart, nil
}

func (cart *DevCard) MappedBanks() string {
	return "-"
}

func (cart *DevCard) ID() string {
	return DevCardID
}

func (cart *DevCard) Snapshot() mapper.CartMapper {
	n := *cart
	return &n

}
func (cart *DevCard) Plumb(env *environment.Environment) {
	cart.env = env
}

func (cart *DevCard) Reset() error {
	return nil
}

func (cart *DevCard) mapping(addr uint16) (int, bool) {
	if addr&0x1000 == 0x1000 && (addr&0x8000 == 0x8000 || addr&0x4000 == 0x4000) {
		b := (((addr & 0xf000) >> 12) - 5) >> 1
		idx := (b * 0x1000) + (addr & 0x0fff)
		return int(idx), true
	}
	return 0, false
}

func (cart *DevCard) Access(addr uint16, peek bool) (data uint8, mask uint8, err error) {
	if idx, ok := cart.mapping(addr); ok {
		return cart.mem[idx], mapper.CartDrivenPins, nil
	}
	return 0, 0, nil
}

func (cart *DevCard) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if idx, ok := cart.mapping(addr); ok {
		cart.mem[idx] = data
	}
	return nil
}

func (cart *DevCard) NumBanks() int {
	return 1
}

func (cart *DevCard) GetBank(addr uint16) banking.Information {
	if _, ok := cart.mapping(addr); ok {
		return banking.Information{Number: 0}
	}
	return banking.Information{Name: "illegal devcart address", NonCart: true}
}

func (cart *DevCard) AccessPassive(addr uint16, data uint8) error {
	return nil
}

func (cart *DevCard) Step(clock float32) {
}

func (cart *DevCard) CopyBanks() []banking.Content {
	c := make([]banking.Content, cart.NumBanks())
	c[0].Number = 0
	c[0].Origins = []uint16{0x5000}
	c[0].Data = make([]uint8, 0xb000)
	copy(c[0].Data[0x0000:0x1000], cart.mem[0x0000:0x1000])
	copy(c[0].Data[0x2000:0x3000], cart.mem[0x1000:0x2000])
	copy(c[0].Data[0x4000:0x5000], cart.mem[0x2000:0x3000])
	copy(c[0].Data[0x6000:0x7000], cart.mem[0x3000:0x4000])
	copy(c[0].Data[0x8000:0x9000], cart.mem[0x4000:0x5000])
	copy(c[0].Data[0xa000:0xb000], cart.mem[0x5000:0x6000])
	return c
}

// ROMDump implements the mapper.CartROMDump interface
func (cart *DevCard) ROMDump(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("%s: %w", DevCardID, err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf(cart.env, "cartridge", "%s: %v", DevCardID, err)
		}
	}()

	_, err = f.Write(cart.mem)
	if err != nil {
		return fmt.Errorf("%s: %w", DevCardID, err)
	}

	return nil
}

// AddressBits implements the CartDevBus interface
func (cart *DevCard) AddressBits() uint16 {
	return 0xffff
}

// ReadWriteLine implements the CartDevBus interface
func (cart *DevCard) ReadWriteLine() bool {
	return true
}
