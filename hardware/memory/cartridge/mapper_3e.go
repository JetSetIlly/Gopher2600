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
	"strings"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

type m3e struct {
	env *environment.Environment

	mappingID string

	bankSize int
	banks    [][]uint8

	// rewindable state
	state *m3eState

	// !!TODO: hotspot info for 3e
}

// cartridges:
//   - Sokoboo
func new3e(env *environment.Environment) (mapper.CartMapper, error) {
	data, err := io.ReadAll(env.Loader)
	if err != nil {
		return nil, fmt.Errorf("F4: %w", err)
	}

	cart := &m3e{
		env:       env,
		mappingID: "3E",
		bankSize:  2048,
		state:     newM3eState(),
	}

	if len(data)%cart.bankSize != 0 {
		return nil, fmt.Errorf("3E: wrong number of bytes in the cartridge data")
	}

	numBanks := len(data) / cart.bankSize
	if numBanks > 255 {
		return nil, fmt.Errorf("3E: too many banks (%d)", numBanks)
	}
	cart.banks = make([][]uint8, numBanks)

	// partition binary
	for k := 0; k < numBanks; k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *m3e) MappedBanks() string {
	s := strings.Builder{}
	s.WriteString("segments: ")
	for i := range cart.state.segment {
		s.WriteString(fmt.Sprintf("%d", cart.state.segment[i]))
		if cart.state.segmentIsRAM[i] {
			s.WriteString("R ")
		} else {
			s.WriteString(" ")
		}
	}
	return strings.TrimSpace(s.String())
}

// ID implements the mapper.CartMapper interface.
func (cart *m3e) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *m3e) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *m3e) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *m3e) Reset() error {
	for b := range cart.state.ram {
		for i := range cart.state.ram[b] {
			if cart.env.Prefs.RandomState.Get().(bool) {
				cart.state.ram[b][i] = uint8(cart.env.Random.Intn(0xff))
			} else {
				cart.state.ram[b][i] = 0
			}
		}
	}

	cart.SetBank("AUTO")

	return nil
}

// Access implements the mapper.CartMapper interface.
func (cart *m3e) Access(addr uint16, _ bool) (uint8, uint8, error) {
	var data uint8

	if addr <= 0x07ff {
		if cart.state.segmentIsRAM[0] {
			if addr <= 0x3ff {
				bank := cart.state.segment[0]
				data = cart.state.ram[bank][addr&0x03ff]
			} else {
				return 0, 0, nil
			}
		} else {
			bank := cart.state.segment[0]
			data = cart.banks[bank][addr&0x07ff]
		}
	} else if addr >= 0x0800 && addr <= 0x0fff {
		bank := cart.state.segment[1]
		data = cart.banks[bank][addr&0x07ff]
	}

	return data, mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *m3e) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if addr <= 0x07ff {
		bank := cart.state.segment[0]
		if cart.state.segmentIsRAM[0] == false {
			if poke {
				cart.banks[bank][addr&0x07ff] = data
			}
			return nil
		}
		if addr >= 0x400 {
			cart.state.ram[bank][addr&0x03ff] = data
		}
	}

	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *m3e) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the mapper.CartMapper interface.
func (cart *m3e) GetBank(addr uint16) mapper.BankInfo {
	if addr >= 0x0000 && addr <= 0x07ff {
		return mapper.BankInfo{Number: cart.state.segment[0], IsRAM: cart.state.segmentIsRAM[0], IsSegmented: true, Segment: 0}
	}
	return mapper.BankInfo{Number: cart.state.segment[1], IsRAM: cart.state.segmentIsRAM[1], IsSegmented: true, Segment: 1}
}

// SetBank implements the mapper.CartMapper interface.
func (cart *m3e) SetBank(bank string) error {
	if mapper.IsAutoBankSelection(bank) {
		// the last segment always points to the last bank
		cart.state.segment[0] = cart.NumBanks() - 2
		cart.state.segment[1] = cart.NumBanks() - 1
		return nil
	}

	segs, err := mapper.SegmentedBankSelection(bank)
	if err != nil {
		return fmt.Errorf("%s: %w", cart.mappingID, err)
	}

	if len(segs) > len(cart.state.segment) {
		return fmt.Errorf("%s: too many segments specified (%d)", cart.mappingID, len(segs))
	}

	b := segs[0]
	if b.Number >= len(cart.banks) {
		return fmt.Errorf("%s: cartridge does not have bank '%d'", cart.mappingID, b.Number)
	}
	cart.state.segment[0] = b.Number
	cart.state.segmentIsRAM[0] = b.IsRAM

	if len(segs) > 1 {
		b = segs[1]
		if b.IsRAM {
			if b.Number >= len(cart.state.ram) {
				return fmt.Errorf("%s: cartridge does not have RAM bank '%d'", cart.mappingID, b.Number)
			}
			cart.state.segment[1] = b.Number
			cart.state.segmentIsRAM[1] = true
		} else {
			if b.Number >= len(cart.banks) {
				return fmt.Errorf("%s: cartridge does not have bank '%d'", cart.mappingID, b.Number)
			}
			cart.state.segment[1] = b.Number
			cart.state.segmentIsRAM[1] = false
		}
	}

	return nil
}

// Patch implements the mapper.CartPatchable interface
func (cart *m3e) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return fmt.Errorf("3E: patch offset too high (%d)", offset)
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *m3e) AccessPassive(addr uint16, data uint8) error {
	// mapper 3e is a derivative of tigervision and so uses the same
	// AccessPassive() mechanism. see the tigervision commentary for details

	// bankswitch on hotspot access
	switch addr {
	case 0x3f:
		romBank := (data & 0x3f) % uint8(cart.NumBanks())
		cart.state.segment[0] = int(romBank)
		cart.state.segmentIsRAM[0] = false
	case 0x3e:
		ramBank := (data & 0x3f) % uint8(len(cart.state.ram))
		cart.state.segment[0] = int(ramBank)
		cart.state.segmentIsRAM[0] = true
	}

	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *m3e) Step(_ float32) {
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart *m3e) GetRAM() []mapper.CartRAM {
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

// CopyBanks implements the mapper.CartMapper interface.
func (cart *m3e) CopyBanks() []mapper.BankContent {
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

// Snapshot implements the mapper.CartMapper interface.
func (s *m3eState) Snapshot() *m3eState {
	n := *s
	for k := 0; k < len(s.ram); k++ {
		n.ram[k] = make([]uint8, len(s.ram[k]))
		copy(n.ram[k], s.ram[k])
	}
	return &n
}
