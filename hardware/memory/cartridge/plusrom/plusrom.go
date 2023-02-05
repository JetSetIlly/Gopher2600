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
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
)

// Sentinal error indicating a specific problem with the attempt to load the
// child cartridge into the PlusROM.
const NotAPlusROM = "not a plus rom: %s"

// PlusROM wraps another mapper.CartMapper inside a network aware format.
type PlusROM struct {
	instance *instance.Instance

	notificationHook notifications.NotificationHook

	net   *network
	state *state

	// rewind boundary is indicated on every network activity
	rewindBoundary bool
}

// rewindable state for the 3e cartridge.
type state struct {
	child mapper.CartMapper
}

// Snapshot implements the mapper.CartMapper interface.
func (s *state) Snapshot() *state {
	n := *s
	n.child = s.child.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (s *state) Plumb() {
	s.child.Plumb()
}

func NewPlusROM(instance *instance.Instance, child mapper.CartMapper, notificationHook notifications.NotificationHook) (mapper.CartMapper, error) {
	cart := &PlusROM{instance: instance}
	cart.notificationHook = notificationHook
	cart.state = &state{}
	cart.state.child = child

	cart.net = newNetwork(cart.instance)

	// get reference to last bank
	bank := child.CopyBanks()[cart.NumBanks()-1]

	// host/path information is found at address 0x1ffa. we've got a reference
	// to the last bank above but we need to consider that the last bank might not
	// take the entirity of the cartridge map.
	addrMask := uint16(len(bank.Data) - 1)

	// host/path information are found at the address pointed to by the following
	// 16bit address
	const addrinfoMSB = 0x1ffb
	const addrinfoLSB = 0x1ffa

	a := uint16(bank.Data[addrinfoLSB&addrMask])
	a |= (uint16(bank.Data[addrinfoMSB&addrMask]) << 8)

	// get bank to which the NMI vector points
	b := int((a & 0xf000) >> 12)

	if b == 0 || b > cart.NumBanks() {
		return nil, curated.Errorf(NotAPlusROM, "invalid NMI vector")
	}

	// normalise indirect address so it's suitable for indexing bank data
	a &= addrMask

	// get the bank to which the NMI vector points
	bank = child.CopyBanks()[b-1]

	// read path string from the first bank using the indirect address retrieved above
	path := strings.Builder{}
	for path.Len() < maxPathLength {
		if int(a) >= len(bank.Data) {
			a = 0x0000
		}
		c := bank.Data[a]

		a++
		if c == 0x00 {
			break // for loop
		}
		path.WriteRune(rune(c))
	}

	// read host string. this string continues on from the path string. the
	// address pointer will be in the correct place.
	host := strings.Builder{}
	for host.Len() <= maxHostLength {
		if int(a) >= len(bank.Data) {
			a = 0x0000
		}
		c := bank.Data[a]

		a++
		if c == 0x00 {
			break // for loop
		}
		host.WriteRune(rune(c))
	}

	// fail if host or path is not valid
	hostValid, pathValid := cart.SetAddrInfo(host.String(), path.String())
	if !hostValid || !pathValid {
		return nil, curated.Errorf(NotAPlusROM, "invalid host/path")
	}

	// log success
	logger.Logf("plusrom", "will connect to %s", cart.net.ai.String())

	// call notificationHook function if one is available
	if cart.notificationHook != nil {
		err := cart.notificationHook(cart, notifications.NotifyPlusROMInserted)
		if err != nil {
			return nil, curated.Errorf("plusrom %v:", err)
		}
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *PlusROM) MappedBanks() string {
	return cart.state.child.MappedBanks()
}

// ID implements the mapper.CartMapper interface.
func (cart *PlusROM) ID() string {
	// not altering the underlying cartmapper's ID
	return cart.state.child.ID()
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *PlusROM) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *PlusROM) Plumb() {
	cart.state.Plumb()
}

// ID implements the mapper.CartContainer interface.
func (cart *PlusROM) ContainerID() string {
	return "PlusROM"
}

// Reset implements the mapper.CartMapper interface.
func (cart *PlusROM) Reset() {
	cart.state.child.Reset()
}

// READ implements the mapper.CartMapper interface.
func (cart *PlusROM) Access(addr uint16, peek bool) (data uint8, mask uint8, err error) {
	switch addr {
	case 0x0ff2:
		// 1FF2 contains the next byte of the response from the host, every
		// read will increment the receive buffer pointer (receive buffer is
		// max 256 bytes also!)
		cart.rewindBoundary = true
		return cart.net.recv(), mapper.CartDrivenPins, nil

	case 0x0ff3:
		// 1FF3 contains the number of (unread) bytes left in the receive buffer
		// (these bytes can be from multiple responses)
		return uint8(cart.net.recvRemaining()), mapper.CartDrivenPins, nil
	}

	return cart.state.child.Access(addr, peek)
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *PlusROM) AccessVolatile(addr uint16, data uint8, poke bool) error {
	switch addr {
	case 0x0ff0:
		// 1FF0 is for writing a byte to the send buffer (max 256 bytes)
		cart.net.buffer(data)
		return nil

	case 0x0ff1:
		// 1FF1 is for writing a byte to the send buffer and submit the buffer
		// to the back end API
		cart.rewindBoundary = true
		cart.net.buffer(data)
		cart.net.commit()
		err := cart.notificationHook(cart, notifications.NotifyPlusROMNetwork)
		if err != nil {
			return curated.Errorf("plusrom %v:", err)
		}
		return nil
	}

	return cart.state.child.AccessVolatile(addr, data, poke)
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *PlusROM) NumBanks() int {
	return cart.state.child.NumBanks()
}

// GetBank implements the mapper.CartMapper interface.
func (cart *PlusROM) GetBank(addr uint16) mapper.BankInfo {
	return cart.state.child.GetBank(addr)
}

// Patch implements the mapper.CartMapper interface.
func (cart *PlusROM) Patch(offset int, data uint8) error {
	return cart.state.child.Patch(offset, data)
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *PlusROM) AccessPassive(addr uint16, data uint8) {
	cart.state.child.AccessPassive(addr, data)
}

// Step implements the mapper.CartMapper interface.
func (cart *PlusROM) Step(clock float32) {
	cart.net.transmitWait()
	cart.state.child.Step(clock)
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *PlusROM) CopyBanks() []mapper.BankContent {
	return cart.state.child.CopyBanks()
}

// GetGetRegisters implements the mapper.CartRegistersBus interface.
func (cart *PlusROM) GetRegisters() mapper.CartRegisters {
	if rb, ok := cart.state.child.(mapper.CartRegistersBus); ok {
		return rb.GetRegisters()
	}
	return nil
}

// PutRegister implements the mapper.CartRegistersBus interface.
func (cart *PlusROM) PutRegister(register string, data string) {
	if rb, ok := cart.state.child.(mapper.CartRegistersBus); ok {
		rb.PutRegister(register, data)
	}
}

// GetRAM implements the mapper.CartRAMbus interface.
func (cart *PlusROM) GetRAM() []mapper.CartRAM {
	if rb, ok := cart.state.child.(mapper.CartRAMbus); ok {
		return rb.GetRAM()
	}
	return nil
}

// PutRAM implements the mapper.CartRAMbus interface.
func (cart *PlusROM) PutRAM(bank int, idx int, data uint8) {
	if rb, ok := cart.state.child.(mapper.CartRAMbus); ok {
		rb.PutRAM(bank, idx, data)
	}
}

// GetStatic implements the mapper.CartStaticBus interface.
func (cart *PlusROM) GetStatic() mapper.CartStatic {
	if sb, ok := cart.state.child.(mapper.CartStaticBus); ok {
		return sb.GetStatic()
	}
	return nil
}

// PutStatic implements the mapper.CartStaticBus interface.
func (cart *PlusROM) PutStatic(segment string, idx int, data uint8) bool {
	if sb, ok := cart.state.child.(mapper.CartStaticBus); ok {
		return sb.PutStatic(segment, idx, data)
	}
	return true
}

// Rewind implements the mapper.CartTapeBus interface.
func (cart *PlusROM) Rewind() {
	if sb, ok := cart.state.child.(mapper.CartTapeBus); ok {
		sb.Rewind()
	}
}

// GetTapeState implements the mapper.CartTapeBus interface.
func (cart *PlusROM) GetTapeState() (bool, mapper.CartTapeState) {
	if sb, ok := cart.state.child.(mapper.CartTapeBus); ok {
		return sb.GetTapeState()
	}
	return false, mapper.CartTapeState{}
}

// RewindBoundary implements the mapper.CartRewindBoundary interface.
func (cart *PlusROM) RewindBoundary() bool {
	if cart.rewindBoundary {
		cart.rewindBoundary = false
		return true
	}
	return false
}
