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

package shim

import (
	"errors"
	"fmt"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
	"go.bug.st/serial"
)

// Shim implements the mapper.CartMapper interface
type Shim struct {
	port serial.Port
	buf  []byte
}

func NewShim() (*Shim, error) {
	var cart Shim
	var err error

	mode := &serial.Mode{
		BaudRate: 57600,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	cart.port, err = serial.Open("/dev/ttyUSB0", mode)
	if err != nil {
		return nil, fmt.Errorf("shim: %w", err)
	}
	cart.buf = make([]byte, 1)

	return &cart, nil
}

// MappedBanks implements the mapper.CartMapper interface
func (cart *Shim) MappedBanks() string {
	return "-"
}

// ID implements the mapper.CartMapper interface
func (cart *Shim) ID() string {
	return "shim"
}

// Snapshot implements the mapper.CartMapper interface
func (cart *Shim) Snapshot() mapper.CartMapper {
	return &Shim{}
}

// Plumb implements the mapper.CartMapper interface
func (cart *Shim) Plumb(_ *environment.Environment) {
}

// Reset implements the mapper.CartMapper interface
func (cart *Shim) Reset() {
}

// Access implements the mapper.CartMapper interface
func (cart *Shim) Access(addr uint16, peek bool) (uint8, uint8, error) {
	addr |= 0x1000
	b, err := cart.updateShim(addr, 0, false)
	if err != nil {
		return 0, 0, fmt.Errorf("shim: access: %w", err)
	}

	// return undriven pins
	return b, mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface
func (cart *Shim) AccessVolatile(addr uint16, data uint8, poke bool) error {
	addr |= 0x1000
	_, err := cart.updateShim(addr, data, true)
	if err != nil {
		return fmt.Errorf("shim: access volatile: %w", err)
	}
	return nil
}

// NumBanks implements the mapper.CartMapper interface
func (cart *Shim) NumBanks() int {
	return 1
}

// GetBank implements the mapper.CartMapper interface
func (cart *Shim) GetBank(_ uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: 0, IsRAM: false}
}

// AccessPassive implements the mapper.CartMapper interface
func (cart *Shim) AccessPassive(addr uint16, data uint8) {
	_, err := cart.updateShim(addr, data, false)
	if err != nil {
		logger.Logf("cartridge", "shim: access passive: %s", err)
	}
}

// Step implements the mapper.CartMapper interface
func (cart *Shim) Step(_ float32) {
}

// Patch implements the mapper.CartMapper interface
func (cart *Shim) Patch(_ int, _ uint8) error {
	return nil
}

// IterateBank implements the mapper.CartMapper interface
func (cart *Shim) CopyBanks() []mapper.BankContent {
	return nil
}

func (cart *Shim) readByte() (uint8, error) {
	n, err := cart.port.Read(cart.buf)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, errors.New("data unavailable")
	}
	return cart.buf[0], nil
}

func (cart *Shim) writeByte(b uint8) error {
	n, err := cart.port.Write([]byte{b})
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("unexpected number of bytes written to serial device")
	}
	return nil
}

func (cart *Shim) updateShim(addr uint16, data uint8, updateData bool) (uint8, error) {
	err := cart.writeByte(byte(addr >> 8))
	if err != nil {
		return 0, err
	}

	err = cart.writeByte(byte(addr))
	if err != nil {
		return 0, err
	}

	if updateData {
		err = cart.writeByte(byte(0xff))
		if err != nil {
			return 0, err
		}

		err = cart.writeByte(byte(data))
		if err != nil {
			return 0, err
		}
	} else {
		err = cart.writeByte(byte(0x00))
		if err != nil {
			return 0, err
		}
	}

	b, err := cart.readByte()
	if err != nil {
		return 0, err
	}

	var checkAddr uint16
	checkAddr = uint16(b) << 8

	b, err = cart.readByte()
	if err != nil {
		return 0, err
	}

	checkAddr |= uint16(b)

	if checkAddr != addr {
		return 0, fmt.Errorf("shim has returned data for an unexpected address: (wanted %04x, got %04x)\n", addr, checkAddr)
	}

	b, err = cart.readByte()
	if err != nil {
		return 0, err
	}

	return b, nil
}
