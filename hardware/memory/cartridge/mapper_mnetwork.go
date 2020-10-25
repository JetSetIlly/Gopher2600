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

// from bankswitch_sizes.txt:
//
// -E7: Only M-Network used this scheme.  This has to be the most complex
// method used in any cart! :-) It allows for the capability of 2K of RAM;
// although it doesn't have to be used (in fact, only one cart used it-
// Burgertime).  This is similar to the 3F type with a few changes.  There are
// now 8 2K banks, instead of 4.
//
// The last 2K in the cart always points to the last 2K of the ROM image, while
// the first 2K is selectable.  You access 1FE0 to 1FE6 to select which 2K
// bank. Note that you cannot select the last 2K of the ROM image into the
// lower 2K of the cart!
//
// Accessing 1FE7 selects 1K of RAM at 1000-17FF instead of ROM!  The 2K of RAM
// is broken up into two 1K sections.  One 1K section is mapped in at 1000-17FF
// if 1FE7 has been accessed.  1000-13FF is the write port, while 1400-17FF is
// the read port.
//
// The second 1K of RAM appears at 1800-19FF.  1800-18FF is the
// write port while 1900-19FF is the read port.  You select which 256 byte
// block appears here by accessing 1FF8 to 1FFB.
//
//
// from the same document, more detail about M-Network RAM:
//
// OK, the RAM setup in these carts is very complex.  There is a total of 2K
// of RAM broken up into 2 1K pieces.  One 1K piece goes into 1000-17FF
// if the bankswitch is set to $1FE7.  The other is broken up into 4 256-byte
// parts.
//
// You select which part to use by issuing a fake read to 1FE8-1FEB.  The
// RAM is then available for use by all banks at 1800-19FF.
//
// Similar to other schemes, 1800-18FF is write while 1900-19FF is read.
// Low RAM uses 1000-13FF for write and 1400-17FF for read.
//
// Note that the 256-byte banks and the large 1K bank are separate entities.
// The M-Network carts are about as complex as it gets.
//
//
// cartridges:
//	- He Man
//	- Pitkat

const num256ByteRAMbanks = 4

type mnetwork struct {
	mappingID   string
	description string

	// mnetwork cartridges have 8 banks of 2048 bytes
	bankSize int
	banks    [][]uint8

	state *mnetworkState
}

func newMnetwork(data []byte) (mapper.CartMapper, error) {
	cart := &mnetwork{
		description: "mnetwork",
		mappingID:   "E7",
		bankSize:    2048,
		state:       newMnetworkState(),
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, curated.Errorf("E7: %v", "wrong number bytes in the cartridge data")
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

func (cart mnetwork) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s [%s]", cart.mappingID, cart.description))
	s.WriteString(fmt.Sprintf(" Bank: %d [%d] ", cart.state.bank, len(cart.banks)-1))
	s.WriteString(fmt.Sprintf(" RAM: %d", cart.state.ram256byteIdx))
	if cart.state.use1kRAM {
		s.WriteString(" [+1k RAM]")
	}
	return s.String()
}

// ID implements the mapper.CartMapper interface.
func (cart mnetwork) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *mnetwork) Snapshot() mapper.CartSnapshot {
	return cart.state.Snapshot()
}

// Plumb implements the mapper.CartMapper interface.
func (cart *mnetwork) Plumb(s mapper.CartSnapshot) {
	cart.state = s.(*mnetworkState)
}

// Reset implements the mapper.CartMapper interface.
func (cart *mnetwork) Reset(randSrc *rand.Rand) {
	for b := range cart.state.ram256byte {
		for i := range cart.state.ram256byte[b] {
			if randSrc != nil {
				cart.state.ram256byte[b][i] = uint8(randSrc.Intn(0xff))
			} else {
				cart.state.ram256byte[b][i] = 0
			}
		}
	}

	for i := range cart.state.ram1k {
		if randSrc != nil {
			cart.state.ram1k[i] = uint8(randSrc.Intn(0xff))
		} else {
			cart.state.ram1k[i] = 0
		}
	}

	cart.state.bank = 0
	cart.state.ram256byteIdx = 0
}

// Read implements the mapper.CartMapper interface.
func (cart *mnetwork) Read(addr uint16, passive bool) (uint8, error) {
	var data uint8

	if addr >= 0x0000 && addr <= 0x07ff {
		if cart.state.use1kRAM && addr >= 0x0400 {
			data = cart.state.ram1k[addr&0x03ff]
		} else {
			data = cart.banks[cart.state.bank][addr&0x07ff]
		}
	} else if addr >= 0x0800 && addr <= 0x0fff {
		if addr >= 0x0900 && addr <= 0x09ff {
			// access upper 1k of ram if cart.segment is pointing to ram and
			// the address is in the write range
			data = cart.state.ram256byte[cart.state.ram256byteIdx][addr&0x00ff]
		} else {
			// if address is not in ram space then read from the last rom bank
			data = cart.banks[cart.NumBanks()-1][addr&0x07ff]
		}
	} else {
		return 0, curated.Errorf("E7: %v", curated.Errorf(bus.AddressError, addr))
	}

	cart.bankswitch(addr, passive)

	return data, nil
}

// Write implements the mapper.CartMapper interface.
func (cart *mnetwork) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if cart.bankswitch(addr, passive) {
		return nil
	}

	if addr >= 0x0000 && addr <= 0x07ff {
		if addr <= 0x03ff && cart.state.use1kRAM {
			cart.state.ram1k[addr&0x03ff] = data
			return nil
		}
	} else if addr >= 0x0800 && addr <= 0x08ff {
		cart.state.ram256byte[cart.state.ram256byteIdx][addr&0x00ff] = data
		return nil
	}

	if poke {
		cart.banks[cart.state.bank][addr] = data
		return nil
	}

	return curated.Errorf("E7: %v", curated.Errorf(bus.AddressError, addr))
}

// bankswitch on hotspot access.
func (cart *mnetwork) bankswitch(addr uint16, passive bool) bool {
	if addr >= 0xfe0 && addr <= 0xfeb {
		if passive {
			return true
		}

		switch addr {
		case 0x0fe0:
			cart.state.bank = 0
			cart.state.use1kRAM = false
		case 0x0fe1:
			cart.state.bank = 1
			cart.state.use1kRAM = false
		case 0x0fe2:
			cart.state.bank = 2
			cart.state.use1kRAM = false
		case 0x0fe3:
			cart.state.bank = 3
			cart.state.use1kRAM = false
		case 0x0fe4:
			cart.state.bank = 4
			cart.state.use1kRAM = false
		case 0x0fe5:
			cart.state.bank = 5
			cart.state.use1kRAM = false
		case 0x0fe6:
			cart.state.bank = 6
			cart.state.use1kRAM = false

			// from bankswitch_sizes.txt: "Note that you cannot select the last 2K
			// of the ROM image into the lower 2K of the cart!  Accessing 1FE7
			// selects 1K of RAM at 1000-17FF instead of ROM!"
			//
			// we're using bank number -1 to indicate the use of RAM
		case 0x0fe7:
			cart.state.use1kRAM = true

			// from bankswitch_sizes.txt: "You select which 256 byte block appears
			// here by accessing 1FF8 to 1FFB."
			//
			// "here" refers to the read range 0x0900 to 0x09ff and the write range
			// 0x0800 to 0x08ff
		case 0x0fe8:
			cart.state.ram256byteIdx = 0
		case 0x0fe9:
			cart.state.ram256byteIdx = 1
		case 0x0fea:
			cart.state.ram256byteIdx = 2
		case 0x0feb:
			cart.state.ram256byteIdx = 3
		}

		return true
	}

	return false
}

// NumBanks implements the mapper.CartMapper interface.
func (cart mnetwork) NumBanks() int {
	return 8 // eight banks of 2k
}

// GetBank implements the mapper.CartMapper interface.
func (cart *mnetwork) GetBank(addr uint16) mapper.BankInfo {
	if addr >= 0x0000 && addr <= 0x07ff {
		if cart.state.use1kRAM {
			return mapper.BankInfo{Number: cart.state.bank, IsRAM: true, Segment: 0}
		}
		return mapper.BankInfo{Number: cart.state.bank, IsRAM: false, Segment: 0}
	}

	if addr >= 0x0800 && addr <= 0x08ff {
		return mapper.BankInfo{Number: cart.state.ram256byteIdx, IsRAM: true, Segment: 1}
	}

	return mapper.BankInfo{Number: 7, IsRAM: false, Segment: 1}
}

// Patch implements the mapper.CartMapper interface.
func (cart *mnetwork) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return curated.Errorf("E7: %v", fmt.Errorf("patch offset too high (%v)", offset))
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the mapper.CartMapper interface.
func (cart *mnetwork) Listen(_ uint16, _ uint8) {
}

// Step implements the mapper.CartMapper interface.
func (cart *mnetwork) Step() {
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart mnetwork) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, num256ByteRAMbanks+1)

	r[0] = mapper.CartRAM{
		Label:  "1k",
		Origin: 0x1000,
		Data:   make([]uint8, len(cart.state.ram1k)),
		Mapped: cart.state.use1kRAM,
	}
	copy(r[0].Data, cart.state.ram1k)

	for i := 0; i < num256ByteRAMbanks; i++ {
		r[i+1] = mapper.CartRAM{
			Label:  fmt.Sprintf("256B [%d]", i),
			Origin: 0x1900,
			Data:   make([]uint8, len(cart.state.ram256byte[i])),
			Mapped: cart.state.ram256byteIdx == i,
		}
		copy(r[i+1].Data, cart.state.ram256byte[i])
	}

	return r
}

// PutRAM implements the mapper.CartRAMBus interface.
func (cart *mnetwork) PutRAM(bank int, idx int, data uint8) {
	if bank == 0 {
		cart.state.ram1k[idx] = data
		return
	}
	cart.state.ram256byte[bank-1][idx] = data
}

// IterateBank implements the mapper.CartMapper interface.
func (cart mnetwork) CopyBanks() []mapper.BankContent {
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

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart mnetwork) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1fe0: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1fe1: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1fe2: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1fe3: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x1fe4: {Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x1fe5: {Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
		0x1fe6: {Symbol: "BANK6", Action: mapper.HotspotBankSwitch},
		0x1fe7: {Symbol: "1kRAM", Action: mapper.HotspotFunction},
		0x1fe8: {Symbol: "RAM0", Action: mapper.HotspotBankSwitch},
		0x1fe9: {Symbol: "RAM1", Action: mapper.HotspotBankSwitch},
		0x1fea: {Symbol: "RAM2", Action: mapper.HotspotBankSwitch},
		0x1feb: {Symbol: "RAM3", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart mnetwork) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}

type mnetworkState struct {
	// identifies the currently selected bank
	bank int

	ram256byte    [num256ByteRAMbanks][]uint8
	ram256byteIdx int

	//  o ram1k is read through addresses 0x1000 to 0x13ff and written
	//  through addresses 0x1400 to 0x17ff * when use1kRAM is true
	//
	//  o ram256byte is read through addresses 0x1900 to 0x19fd and written
	//  through address 0x1800 to 0x18ff in all cases
	//
	// (addresses quoted above are of course masked so that they fall into the
	// allocation range)
	ram1k []uint8

	// use1kRAM is set to true when hotspot 0x0fe7 has been triggered. it's not
	// clear when, if ever, the flag should be set to false. we have taken the
	// view that is is when any of hotspots 0x0fe0 to 0x0fe6 are triggered
	use1kRAM bool
}

func newMnetworkState() *mnetworkState {
	s := &mnetworkState{}

	// not all m-network cartridges have RAM but we'll allocate it for all
	// instances because there's no way of detecting if it does or not.
	for i := range s.ram256byte {
		s.ram256byte[i] = make([]uint8, 256)
	}

	s.ram1k = make([]uint8, 1024)

	return s
}

// Snapshot implements the mapper.CartSnapshot interface.
func (s *mnetworkState) Snapshot() mapper.CartSnapshot {
	n := *s
	for i := range s.ram256byte {
		n.ram256byte[i] = make([]uint8, len(s.ram256byte[i]))
		copy(n.ram256byte[i], s.ram256byte[i])
	}
	n.ram1k = make([]uint8, len(s.ram1k))
	copy(n.ram1k, s.ram1k)
	return &n
}
