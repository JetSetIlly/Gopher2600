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
	"time"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"go.bug.st/serial"
)

// Shim implements the mapper.CartMapper interface
type Shim struct {
	port serial.Port
	buf  []byte

	// killed indicates that an error has occurred with the communication with
	// the cartridge shim and that all subsequent accesses should return a KIL
	// instruction. non-read accesses will do nothing
	killed bool
}

func NewShim() (*Shim, error) {
	cart := Shim{
		buf: make([]byte, 4),
	}

	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	var err error
	cart.port, err = serial.Open("/dev/ttyUSB0", mode)
	if err != nil {
		return nil, fmt.Errorf("shim: %w", err)
	}
	err = cart.port.SetReadTimeout(1 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("shim: %w", err)
	}

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
	if cart.killed {
		return 0x02, mapper.CartDrivenPins, nil
	}

	data, err := cart.update(addr|0x1000, 0x00, false)
	if err != nil {
		cart.killed = true
		return 0x02, mapper.CartDrivenPins, err
	}

	return data, mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface
func (cart *Shim) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if cart.killed {
		return nil
	}

	data, err := cart.update(addr|0x1000, data, true)
	if err != nil {
		cart.killed = true
		return err
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
func (cart *Shim) AccessPassive(addr uint16, data uint8) error {
	if cart.killed {
		return nil
	}

	data, err := cart.update(addr|0x1000, data, true)
	if err != nil {
		cart.killed = true
		return err
	}

	return nil
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

func (cart *Shim) readBytes(port serial.Port, buf []byte) error {
	n, err := port.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("data unavailable")
	}
	return nil
}

func (cart *Shim) writeBytes(port serial.Port, buf []uint8) error {
	n, err := port.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return errors.New("unexpected number of bytes written to serial device")
	}
	return nil
}

func (cart *Shim) update(addr uint16, data uint8, updateData bool) (uint8, error) {
	var err error
	if updateData {
		cart.buf[0] = 0xff
		cart.buf[1] = data
		cart.buf[2] = byte(addr >> 8)
		cart.buf[3] = byte(addr)
		err = cart.writeBytes(cart.port, cart.buf[:4])
	} else {
		cart.buf[0] = 0x00
		cart.buf[1] = byte(addr >> 8)
		cart.buf[2] = byte(addr)
		err = cart.writeBytes(cart.port, cart.buf[:3])
	}
	if err != nil {
		return 0, fmt.Errorf("shim: %w", err)
	}

	err = cart.readBytes(cart.port, cart.buf[:3])
	if err != nil {
		return 0, fmt.Errorf("shim: %w", err)
	}

	var checkAddr uint16
	checkAddr = (uint16(cart.buf[0]) << 8) | uint16(cart.buf[1])

	if checkAddr != addr {
		return 0, fmt.Errorf("unexpected address returned by cartridge shim: wanted %04x, got %04x", addr, checkAddr)
	}

	return cart.buf[2], nil
}
