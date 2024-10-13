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

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// from bankswitch_sizes.txt:
//
// -3F: Tigervision was the only user of this intresting method.  This works
// in a similar fashion to the above method; however, there are only 4 2K
// segments instead of 4 1K ones, and the ROM image is broken up into 4 2K
// slices.  As before, the last 2K always points to the last 2K of the image.
// You select the desired bank by performing an STA $3F instruction.  The
// accumulator holds the desired bank number (0-3; only the lower two bits are
// used).  Any STA in the $00-$3F range will change banks.  This appears to
// interfere with the TIA addresses, which it does; however you just use $40 to
// $7F instead! :-)  $3F does not have a corresponding TIA register, so writing
// here has no effect other than switching banks.  Very clever; especially
// since you can implement this with only one chip! (a 74LS173).
//
// cartridges:
//   - Miner2049
//   - River Patrol
type tigervision struct {
	env *environment.Environment

	mappingID string

	// tigervision cartridges traditionally have 4 of banks of 2048 bytes. but
	// it can theoretically support anything up to 512 banks
	bankSize int
	banks    [][]uint8

	// rewindable state
	state *tigervisionState

	// !!TODO: hotspot info for tigervision
}

// should work with any size cartridge that is a multiple of 2048:
//   - tested with 8k (Miner2049 etc.) and 32k (Genesis_Egypt demo).
func newTigervision(env *environment.Environment, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	data, err := io.ReadAll(loader)
	if err != nil {
		return nil, fmt.Errorf("3F: %w", err)
	}

	cart := &tigervision{
		env:       env,
		mappingID: "3F",
		bankSize:  2048,
		state:     newTigervisionState(),
	}

	if len(data)%cart.bankSize != 0 {
		return nil, fmt.Errorf("3F: wrong number of bytes in the cartridge data")
	}

	numBanks := len(data) / cart.bankSize
	cart.banks = make([][]uint8, numBanks)

	for k := 0; k < numBanks; k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *tigervision) MappedBanks() string {
	return fmt.Sprintf("Banks: %d %d", cart.state.segment[0], cart.state.segment[1])
}

// ID implements the mapper.CartMapper interface.
func (cart *tigervision) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *tigervision) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *tigervision) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *tigervision) Reset() {
	cart.SetBank("AUTO")
}

// Access implements the mapper.CartMapper interface.
func (cart *tigervision) Access(addr uint16, _ bool) (uint8, uint8, error) {
	var data uint8
	if addr >= 0x0000 && addr <= 0x07ff {
		data = cart.banks[cart.state.segment[0]][addr&0x07ff]
	} else if addr >= 0x0800 && addr <= 0x0fff {
		data = cart.banks[cart.state.segment[1]][addr&0x07ff]
	}
	return data, mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *tigervision) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if poke {
		if addr >= 0x0000 && addr <= 0x07ff {
			cart.banks[cart.state.segment[0]][addr&0x07ff] = data
		} else if addr >= 0x0800 && addr <= 0x0fff {
			cart.banks[cart.state.segment[1]][addr&0x07ff] = data
		}
	}
	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *tigervision) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the mapper.CartMapper interface.
func (cart *tigervision) GetBank(addr uint16) mapper.BankInfo {
	if addr >= 0x0000 && addr <= 0x07ff {
		return mapper.BankInfo{Number: cart.state.segment[0], IsRAM: false, IsSegmented: true, Segment: 0}
	}
	return mapper.BankInfo{Number: cart.state.segment[1], IsRAM: false, IsSegmented: true, Segment: 1}
}

// SetBank implements the mapper.CartMapper interface.
func (cart *tigervision) SetBank(bank string) error {
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

	for i, b := range segs {
		if b.Number >= len(cart.banks) {
			return fmt.Errorf("%s: cartridge does not have bank '%d'", cart.mappingID, b.Number)
		}
		if b.IsRAM {
			return fmt.Errorf("%s: cartridge does not have bankable RAM", cart.mappingID)
		}

		if i == len(cart.state.segment)-1 && b.Number != cart.NumBanks()-1 {
			return fmt.Errorf("%s: last segment must always be bank %d", cart.mappingID, cart.NumBanks()-1)
		}

		cart.state.segment[i] = b.Number
	}

	return nil
}

// Patch implements the mapper.CartPatchable interface.
func (cart *tigervision) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return fmt.Errorf("3F: patch offset too high (%d)", offset)
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *tigervision) AccessPassive(addr uint16, data uint8) error {
	// although address 3F is used primarily, in actual fact writing anywhere
	// in TIA space is okay. from  the description from Kevin Horton's document
	// (quoted above) whenever an address in TIA space is written to, the lower
	// 3 bits of the value being written is used to set the segment.
	//
	// however, taken literally, the foregoing can have issue with phantom
	// reads. for example, Miner2049 with the instruction STA VSYNC,X (bank 2
	// address $3611) will cause a phantom read of $0000 - but this produces
	// incorrect results
	//
	// the following comment provides sufficient details to emulate the scheme
	// correctly:
	//
	// https://atariage.com/forums/topic/329888-indexed-read-page-crossing-and-sc-ram/?do=findComment&comment=4988836
	//
	// alex_79 writes: "The bankswitch happens if any address with both A6 and
	// A7 low is accessed, and if A12 goes from low to high right after that
	// access."

	// A12 is high after being low (bankswitchPending implies that A12 was low
	// on the previous bus transition)
	if cart.state.bankswitchPending && addr&0x1000 == 0x1000 {
		cart.state.segment[0] = int(data & uint8(cart.NumBanks()-1))
	}

	// A6 and A7 is low. A12 must be low also.
	cart.state.bankswitchPending = addr&0x10c0 == 0x0000

	// this bank switching method can cause problems when the CPU wants to
	// write to TIA space for real and not cause a bankswitch. for this reason,
	// tigervision cartridges use mirror addresses to write to the TIA.

	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *tigervision) Step(_ float32) {
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *tigervision) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))

	// banks 0 to len-1 can occupy any of the first three segments
	for b := 0; b < len(cart.banks)-1; b++ {
		c[b] = mapper.BankContent{Number: b,
			Data: cart.banks[b],
			Origins: []uint16{
				memorymap.OriginCart,
				memorymap.OriginCart + uint16(cart.bankSize),
				memorymap.OriginCart + uint16(cart.bankSize)*2,
			},
		}
	}

	// last bank cannot point to the first segment
	b := len(cart.banks) - 1
	c[b] = mapper.BankContent{Number: b,
		Data: cart.banks[b],
		Origins: []uint16{
			memorymap.OriginCart + uint16(cart.bankSize),
			memorymap.OriginCart + uint16(cart.bankSize)*2,
			memorymap.OriginCart + uint16(cart.bankSize)*3,
		},
	}
	return c
}

// rewindable state for the tigervision cartridges.
type tigervisionState struct {
	bankswitchPending bool

	// tigervision cartridges divide memory into two 2k segments
	//  o the last segment always points to the last bank
	//  o the first segment can point to any of the other three
	//
	// the bank pointed to by the first segment is changed through the listen()
	// function (part of the implementation of the mapper.CartMapper interface).
	segment [2]int
}

func newTigervisionState() *tigervisionState {
	return &tigervisionState{}
}

// Snapshot implements the mapper.CartMapper interface.
func (s *tigervisionState) Snapshot() *tigervisionState {
	n := *s
	return &n
}
