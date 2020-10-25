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
	"math/rand"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

type m3e struct {
	mappingID   string
	description string

	bankSize int
	banks    [][]uint8

	// rewindable state
	state *m3eState
}

// cartridges:
//	- Sokoboo
func new3e(data []byte) (mapper.CartMapper, error) {
	cart := &m3e{
		mappingID:   "3E",
		description: "m3e",
		bankSize:    2048,
		state:       newM3eState(),
	}

	if len(data)%cart.bankSize != 0 {
		return nil, curated.Errorf("3E: %v", "wrong number bytes in the cartridge data")
	}

	numBanks := len(data) / cart.bankSize
	cart.banks = make([][]uint8, numBanks)

	// partition binary
	for k := 0; k < numBanks; k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

func (cart m3e) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s segments: ", cart.mappingID))
	for i := range cart.state.segment {
		s.WriteString(fmt.Sprintf("%d", cart.state.segment[i]))
		if cart.state.segmentIsRAM[i] {
			s.WriteString("R ")
		} else {
			s.WriteString(" ")
		}
	}
	return s.String()
}

// ID implements the mapper.CartMapper interface.
func (cart m3e) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *m3e) Snapshot() mapper.CartSnapshot {
	return cart.state.Snapshot()
}

// Plumb implements the mapper.CartMapper interface.
func (cart *m3e) Plumb(s mapper.CartSnapshot) {
	cart.state = s.(*m3eState)
}

// Reset implements the mapper.CartMapper interface.
func (cart *m3e) Reset(randSrc *rand.Rand) {
	for b := range cart.state.ram {
		for i := range cart.state.ram[b] {
			if randSrc != nil {
				cart.state.ram[b][i] = uint8(randSrc.Intn(0xff))
			} else {
				cart.state.ram[b][i] = 0
			}
		}
	}

	// the last segment always points to the last bank
	cart.state.segment[0] = cart.NumBanks() - 2
	cart.state.segment[1] = cart.NumBanks() - 1
}

// Read implements the mapper.CartMapper interface.
func (cart *m3e) Read(addr uint16, _ bool) (uint8, error) {
	var segment int

	if addr >= 0x0000 && addr <= 0x07ff {
		segment = 0
	} else if addr >= 0x0800 && addr <= 0x0fff {
		segment = 1
	}

	var data uint8

	if cart.state.segmentIsRAM[segment] {
		data = cart.state.ram[cart.state.segment[segment]][addr&0x03ff]
	} else {
		bank := cart.state.segment[segment]
		if bank < len(cart.banks) {
			data = cart.banks[bank][addr&0x07ff]
		}
	}

	return data, nil
}

// Write implements the mapper.CartMapper interface.
func (cart *m3e) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if passive {
		return nil
	}

	var segment int

	if addr >= 0x0000 && addr <= 0x07ff {
		segment = 0
	} else if addr >= 0x0c00 && addr <= 0x0fff {
		segment = 3
	}

	if cart.state.segmentIsRAM[segment] {
		cart.state.ram[cart.state.segment[segment]][addr&0x03ff] = data
		return nil
	} else if poke {
		cart.banks[cart.state.segment[segment]][addr&0x07ff] = data
		return nil
	}

	return curated.Errorf("3E: %v", curated.Errorf(bus.AddressError, addr))
}

// NumBanks implements the mapper.CartMapper interface.
func (cart m3e) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the mapper.CartMapper interface.
func (cart *m3e) GetBank(addr uint16) mapper.BankInfo {
	if addr >= 0x0000 && addr <= 0x07ff {
		return mapper.BankInfo{Number: cart.state.segment[0], IsRAM: false, Segment: 0}
	}
	return mapper.BankInfo{Number: cart.state.segment[1], IsRAM: false, Segment: 1}
}

// Patch implements the mapper.CartMapper interface.
func (cart *m3e) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return curated.Errorf("3E: %v", fmt.Errorf("patch offset too high (%v)", offset))
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the mapper.CartMapper interface.
func (cart *m3e) Listen(addr uint16, data uint8) {
	// mapper 3e is a derivative of tigervision and so uses the same Listen()
	// mechanism. see the tigervision commentary for details

	// bankswitch on hotspot access
	if addr == 0x3f {
		segment := data >> 6
		bank := data & 0x3f
		cart.state.segment[segment] = int(bank)
		cart.state.segmentIsRAM[segment] = false
	} else if addr == 0x3e {
		segment := data >> 6
		bank := data & 0x3f
		cart.state.segment[segment] = int(bank)
		cart.state.segmentIsRAM[segment] = true
	}
}

// Step implements the mapper.CartMapper interface.
func (cart *m3e) Step() {
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart m3e) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, len(cart.state.ram))

	for i := range cart.state.ram {
		mapped := false
		origin := uint16(0x0000)

		for s := range cart.state.segment {
			mapped = cart.state.segment[s] == i && cart.state.segmentIsRAM[s]
			if mapped {
				switch s {
				case 0:
					origin = uint16(0x1000)
				case 1:
					origin = uint16(0x1800)
				}
				break // for loop
			}
		}

		r[i] = mapper.CartRAM{
			Label:  fmt.Sprintf("%d", i),
			Origin: origin,
			Data:   make([]uint8, len(cart.state.ram[i])),
			Mapped: mapped,
		}
		copy(r[i].Data, cart.state.ram[i])
	}

	return r
}

// PutRAM implements the mapper.CartRAMBus interface.
func (cart *m3e) PutRAM(bank int, idx int, data uint8) {
	cart.state.ram[bank][idx] = data
}

// IterateBank implements the mapper.CartMapper interface.
func (cart m3e) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))

	for b := 0; b < len(cart.banks)-1; b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}

	// always points to the last segment
	b := len(cart.banks) - 1
	c[b] = mapper.BankContent{Number: b,
		Data:    cart.banks[b],
		Origins: []uint16{memorymap.OriginCart + uint16(cart.bankSize)},
	}
	return c
}

// rewindable state for the 3e cartridge.
type m3eState struct {
	// 32 is the maximum number of banks possible under the 3e scheme
	ram [32][]uint8

	// 3e cartridges divide memory into two 2k segments
	//  o the last segment always points to the last bank
	//  o the first segment can point to any of the other three
	//
	// the bank pointed to by the first segment is changed through the listen()
	// function (part of the implementation of the mapper.CartMapper interface).
	segment [2]int

	// in the 3e format only the first segment can contain RAM but for
	// simplicity we keep track of both segments
	segmentIsRAM [2]bool
}

func newM3eState() *m3eState {
	s := &m3eState{}

	// a ram bank is half the size of the available bank size. this is because
	// of how ram is accessed - half the addresses are for reading and the
	// other half are for writing.
	const ramSize = 1024

	// allocate ram
	for k := 0; k < len(s.ram); k++ {
		s.ram[k] = make([]uint8, ramSize)
	}

	return s
}

// Snapshot implements the mapper.CartSnapshot interface.
func (s *m3eState) Snapshot() mapper.CartSnapshot {
	n := *s

	for k := 0; k < len(s.ram); k++ {
		n.ram[k] = make([]uint8, len(s.ram[k]))
		copy(n.ram[k], s.ram[k])
	}

	return &n
}
