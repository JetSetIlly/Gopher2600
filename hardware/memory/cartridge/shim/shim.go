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

type updateRequest struct {
	addr       uint16
	data       uint8
	updateData bool
}

// Shim implements the mapper.CartMapper interface
type Shim struct {
	upd  chan updateRequest
	ret  chan uint8
	quit chan bool
}

func NewShim() (*Shim, error) {
	cart := Shim{
		upd:  make(chan updateRequest, 32),
		ret:  make(chan uint8, 1),
		quit: make(chan bool, 1),
	}

	setupError := make(chan error)
	go cart.updateShim(setupError)

	err := <-setupError
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
	cart.upd <- updateRequest{
		addr: addr | 0x1000,
	}
	return <-cart.ret, mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface
func (cart *Shim) AccessVolatile(addr uint16, data uint8, poke bool) error {
	cart.upd <- updateRequest{
		addr:       addr | 0x1000,
		data:       data,
		updateData: true,
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
	cart.upd <- updateRequest{
		addr:       addr,
		data:       data,
		updateData: true,
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

func (cart *Shim) updateShim(setupError chan error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open("/dev/ttyUSB0", mode)
	setupError <- err

	buf := make([]byte, 4)
	for {
		select {
		case <-cart.quit:
			return

		case req := <-cart.upd:
			var err error
			if req.updateData {
				buf[0] = byte(req.addr >> 8)
				buf[1] = byte(req.addr)
				buf[2] = 0xff
				buf[3] = req.data
				err = cart.writeBytes(port, buf[:4])
			} else {
				buf[0] = byte(req.addr >> 8)
				buf[1] = byte(req.addr)
				buf[2] = 0x00
				err = cart.writeBytes(port, buf[:3])
			}
			if err != nil {
				logger.Logf("shim", err.Error())
			}

			err = cart.readBytes(port, buf[:3])
			if err != nil {
				logger.Logf("shim", err.Error())
			}

			var checkAddr uint16
			checkAddr = (uint16(buf[0]) << 8) | uint16(buf[1])

			if checkAddr != req.addr {
				logger.Logf("shim", "unexpected address returned by cartridge shim: wanted %04x, got %04x", req.addr, checkAddr)
			}

			if !req.updateData {
				select {
				case <-cart.quit:
					return
				case cart.ret <- buf[2]:
				}
			}
		}
	}
}
