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

package plusrom

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

// PlusROM wraps another mapper.CartMapper inside a network aware format
type PlusROM struct {
	child mapper.CartMapper
	net   network
}

func NewPlusROM(child mapper.CartMapper) (mapper.CartMapper, error) {
	cart := &PlusROM{child: child}

	bank := &banks.Content{Number: -1}
	for i := 0; i < cart.NumBanks(); i++ {
		bank = child.IterateBanks(bank)
	}

	// host address is made up of address 0x1ffa (LSB) and 0x1ffb (MSB)
	// making sure to index the data correctly
	a := uint16(bank.Data[0x0ffa])
	a |= (uint16(bank.Data[0x0ffb]) << 8)

	// normalise address so it's suitable for indexing bank data
	a &= 0x0fff
	a++

	// read path string
	s := strings.Builder{}
	for {
		if int(a) >= len(bank.Data) {
			a = 0x0000
		}
		c := bank.Data[a]
		a++
		if c == 0x00 {
			break // for loop
		}
		s.WriteRune(rune(c))
	}
	cart.net.addr.Path = s.String()

	// read host string
	s.Reset()
	for {
		if int(a) >= len(bank.Data) {
			a = 0x0000
		}
		c := bank.Data[a]
		a++
		if c == 0x00 {
			break // for loop
		}
		s.WriteRune(rune(c))
	}
	cart.net.addr.Host = s.String()

	// log success
	logger.Log("plusrom", fmt.Sprintf("%s/%s", cart.net.addr.Host, cart.net.addr.Path))

	return cart, nil
}

// Initialise implements the mapper.CartMapper interface
func (cart *PlusROM) Initialise() {
	cart.child.Initialise()
}

func (cart *PlusROM) String() string {
	// add PlusROM indicator to String
	return fmt.Sprintf("[%s] %s", cart.ContainerID(), cart.child.String())
}

// ID implements the mapper.CartMapper interface
func (cart *PlusROM) ID() string {
	// not altering the underlying cartmapper's ID
	return cart.child.ID()
}

// ID implements the mapper.CartContainer interface
func (cart *PlusROM) ContainerID() string {
	return "PlusROM"
}

// READ implements the mapper.CartMapper interface
func (cart *PlusROM) Read(addr uint16, active bool) (data uint8, err error) {
	switch addr {
	case 0x0ff2:
		// 1FF2 contains the next byte of the response from the host, every
		// read will increment the receive buffer pointer (receive buffer is
		// max 256 bytes also!)
		return cart.net.recv(), nil

	case 0x0ff3:
		// 1FF3 contains the number of (unread) bytes left in the receive buffer
		// (these bytes can be from multiple responses)
		return uint8(cart.net.recvRemaining()), nil
	}

	return cart.child.Read(addr, active)
}

// Write implements the mapper.CartMapper interface
func (cart *PlusROM) Write(addr uint16, data uint8, active bool, poke bool) error {
	switch addr {
	case 0x0ff0:
		// 1FF0 is for writing a byte to the send buffer (max 256 bytes)
		cart.net.send(data, false)
		return nil

	case 0x0ff1:
		// 1FF1 is for writing a byte to the send buffer and submit the buffer
		// to the back end API
		cart.net.send(data, true)
		return nil
	}

	return cart.child.Write(addr, data, active, poke)
}

// NumBanks implements the mapper.CartMapper interface
func (cart *PlusROM) NumBanks() int {
	return cart.child.NumBanks()
}

// GetBank implements the mapper.CartMapper interface
func (cart *PlusROM) GetBank(addr uint16) banks.Details {
	return cart.child.GetBank(addr)
}

// Patch implements the mapper.CartMapper interface
func (cart *PlusROM) Patch(offset int, data uint8) error {
	return cart.child.Patch(offset, data)
}

// Listen implements the mapper.CartMapper interface
func (cart *PlusROM) Listen(addr uint16, data uint8) {
	cart.child.Listen(addr, data)
}

// Step implements the mapper.CartMapper interface
func (cart *PlusROM) Step() {
	cart.child.Step()
}

// IterateBanks implements the mapper.CartMapper interface
func (cart *PlusROM) IterateBanks(prev *banks.Content) *banks.Content {
	return cart.child.IterateBanks(prev)
}

// GetGetRegisters implements the bus.CartRegistersBus interface
func (cart *PlusROM) GetRegisters() bus.CartRegisters {
	if rb, ok := cart.child.(bus.CartRegistersBus); ok {
		return rb.GetRegisters()
	}
	return nil
}

// PutRegister implements the bus.CartRegistersBus interface
func (cart *PlusROM) PutRegister(register string, data string) {
	if rb, ok := cart.child.(bus.CartRegistersBus); ok {
		rb.PutRegister(register, data)
	}
}

// GetRAM implements the bus.CartRAMbus interface
func (cart *PlusROM) GetRAM() []bus.CartRAM {
	if rb, ok := cart.child.(bus.CartRAMbus); ok {
		return rb.GetRAM()
	}
	return nil
}

// PutRAM implements the bus.CartRAMbus interface
func (cart *PlusROM) PutRAM(bank int, idx int, data uint8) {
	if rb, ok := cart.child.(bus.CartRAMbus); ok {
		rb.PutRAM(bank, idx, data)
	}
}

// GetStatic implements the bus.CartStaticBus interface
func (cart *PlusROM) GetStatic() []bus.CartStatic {
	if sb, ok := cart.child.(bus.CartStaticBus); ok {
		return sb.GetStatic()
	}
	return nil
}

// PutStatic implements the bus.CartStaticBus interface
func (cart *PlusROM) PutStatic(tag string, addr uint16, data uint8) error {
	if sb, ok := cart.child.(bus.CartStaticBus); ok {
		return sb.PutStatic(tag, addr, data)
	}
	return nil
}

// Rewind implements the bus.CartTapeBus interface
func (cart *PlusROM) Rewind() bool {
	if sb, ok := cart.child.(bus.CartTapeBus); ok {
		return sb.Rewind()
	}
	return false
}

// GetTapeState implements the bus.CartTapeBus interface
func (cart *PlusROM) GetTapeState() (bool, bus.CartTapeState) {
	if sb, ok := cart.child.(bus.CartTapeBus); ok {
		return sb.GetTapeState()
	}
	return false, bus.CartTapeState{}
}

// GetNetwork returns a new instance of PlusROMAddrInfo
func (cart *PlusROM) GetNetwork() AddrInfo {
	return AddrInfo{
		Host: cart.net.addr.Host,
		Path: cart.net.addr.Path,
	}
}

// SetNetwork updates the host/path information int the PlusROM
func (cart *PlusROM) SetNetwork(host string, path string) {
	cart.net.addr.Host = host
	cart.net.addr.Path = path
}