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

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// from "Dr Boo's Woodgrain Wizardry"
// http://www.taswegian.com/WoodgrainWizard/tiki-index.php?page=WD

// WD
// 8K ROM
// 64 bytes RAM
//
// The 2600's 4K cartridge address space is broken into four 1K segments. The
// desired arrangement of 1K banks is selected by accessing $30 - $3F of TIA
// address space. The banks are mapped into all 4 segments at once as follows:
//
//     $0030, $0038: 0,0,1,3
//     $0031, $0039: 0,1,2,3
//     $0032, $003A: 4,5,6,7
//     $0033, $003B: 7,4,2,3
//
//     $0034, $003C: 0,0,6,7
//     $0035, $003D: 0,1,7,6
//     $0036, $003E: 2,3,4,5
//     $0037, $003F: 6,0,5,1
//
//
// The 64 bytes of RAM are accessible at $1000 - $103F (read port) and $1040 -
// $107F (write port). Because the RAM takes 128 bytes of address space, the
// range $1000 - $107F of segment 0 ROM will never be available.

// further information was taken from the Stella source code. this was
// necessary because some details are not written anywhere else. for instance,
// the fact that the bankswitch occurs three cpu cycles after the hotspot
// access.
//
// in addition details about the "bad ROM dump" of the Pink Panther was
// originally documented in the Stella source

type wicksteadDesign struct {
	env *environment.Environment

	mappingID string

	// wickstead cartridges have 3 banks of 4096 bytes
	bankSize int
	banks    [][]uint8

	// rewindable state
	state *wicksteadState
}

func newWicksteadDesign(env *environment.Environment) (mapper.CartMapper, error) {
	cart := &wicksteadDesign{
		env:       env,
		mappingID: "WD",
		bankSize:  1024,
		state:     newWicksteadState(),
	}

	data, err := io.ReadAll(env.Loader)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", cart.mappingID, err)
	}

	// the only known ROM that uses this mapper is the prototype of the Pink
	// Panther game. Unfortunately there is a bad dump of the ROM that swaps
	// banks 2 and 3
	//
	// if we detect this badly dumped ROM file we use this flag to swap the two
	// banks
	badDump := false

	if len(data) != cart.bankSize*cart.NumBanks() {
		if len(data) == cart.bankSize*cart.NumBanks()+3 {
			badDump = true
		} else {
			return nil, fmt.Errorf("%s: wrong number of bytes in the cartridge data", cart.mappingID)
		}
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	if badDump {
		cart.banks[2], cart.banks[3] = cart.banks[3], cart.banks[2]
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) MappedBanks() string {
	return fmt.Sprintf("Banks: %d, %d, %d, %d",
		cart.state.segments[0], cart.state.segments[1], cart.state.segments[2], cart.state.segments[3])
}

// ID implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) Reset() error {
	for i := range cart.state.ram {
		if cart.env.Prefs.RandomState.Get().(bool) {
			cart.state.ram[i] = uint8(cart.env.Random.Intn(0xff))
		} else {
			cart.state.ram[i] = 0
		}
	}

	cart.SetBank("AUTO")

	return nil
}

// Access implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) Access(addr uint16, _ bool) (uint8, uint8, error) {
	if addr <= 0x003f {
		return cart.state.ram[addr], mapper.CartDrivenPins, nil
	}
	if addr >= 0x0040 && addr <= 0x007f {
		return 0, 0x0, nil
	}

	_, bank, idx := cart.getBank(addr)
	return cart.banks[bank][idx], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if addr >= 0x0040 && addr <= 0x007f {
		cart.state.ram[addr-0x0040] = data
		return nil
	}

	_, bank, idx := cart.getBank(addr)

	if poke {
		cart.banks[bank][idx] = data
		return nil
	}

	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) NumBanks() int {
	return 8
}

func (cart *wicksteadDesign) getBank(addr uint16) (int, int, uint16) {
	var segment int
	var bank int
	var idx uint16

	if addr <= 0x03ff {
		segment = 0
		bank = cart.state.segments[0]
		idx = addr
	} else if addr >= 0x0400 && addr <= 0x07ff {
		segment = 1
		bank = cart.state.segments[1]
		idx = addr - 0x0400
	} else if addr >= 0x0800 && addr <= 0x0bff {
		segment = 2
		bank = cart.state.segments[2]
		idx = addr - 0x0800
	} else if addr >= 0x0c00 && addr <= 0x0fff {
		segment = 3
		bank = cart.state.segments[3]
		idx = addr - 0x0c00
	}

	return segment, bank, idx
}

// GetBank implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) GetBank(addr uint16) mapper.BankInfo {
	segment, bank, _ := cart.getBank(addr)
	return mapper.BankInfo{
		Number:      bank,
		IsRAM:       addr < 0x003f,
		IsSegmented: true,
		Segment:     segment,
	}
}

// SetBank implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) SetBank(bank string) error {
	if mapper.IsAutoBankSelection(bank) {
		p, err := cart.segmentPattern(0)
		if err != nil {
			return err
		}
		cart.state.segments = p
		return nil
	}

	// wickstead design uses a pattern selector. we can use the single bank
	// selection function for this

	b, err := mapper.SingleBankSelection(bank)
	if err != nil {
		return fmt.Errorf("%s: %w", cart.mappingID, err)
	}
	if b.IsRAM {
		return fmt.Errorf("%s: cartridge expects a pattern number between 0 and 7", cart.mappingID)
	}

	p, err := cart.segmentPattern(b.Number)
	if err != nil {
		return err
	}
	cart.state.segments = p

	return nil
}

// Patch implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return fmt.Errorf("FA: patch offset too high (%d)", offset)
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

func (cart *wicksteadDesign) segmentPattern(pattern int) ([4]int, error) {
	var p [4]int

	switch pattern {
	case 0:
		p = [4]int{0, 0, 1, 3}
	case 1:
		p = [4]int{0, 1, 2, 3}
	case 2:
		p = [4]int{4, 5, 6, 7}
	case 3:
		p = [4]int{7, 4, 2, 3}
	case 4:
		p = [4]int{0, 0, 6, 7}
	case 5:
		p = [4]int{0, 1, 7, 6}
	case 6:
		p = [4]int{2, 3, 4, 5}
	case 7:
		p = [4]int{6, 0, 5, 1}
	default:
		return [4]int{}, fmt.Errorf("%s: invalid segment pattern (%d)", cart.mappingID, pattern)
	}

	return p, nil
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) AccessPassive(addr uint16, data uint8) error {
	// switch bank pattern
	if addr&0xfff0 == 0x0030 {
		pattern := int(addr & 0x0007)
		var err error
		cart.state.pending, err = cart.segmentPattern(pattern)
		if err != nil {
			return err
		}
		cart.state.pendingCt = 4
	}

	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) Step(_ float32) {
	if cart.state.pendingCt > 0 {
		cart.state.pendingCt--
		if cart.state.pendingCt == 0 {
			cart.state.segments = cart.state.pending
		}
	}
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart *wicksteadDesign) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, 1)
	r[0] = mapper.CartRAM{
		Label:  "Wickstead Design",
		Origin: 0x1000,
		Data:   make([]uint8, len(cart.state.ram)),
		Mapped: true,
	}
	copy(r[0].Data, cart.state.ram)
	return r
}

// PutRAM implements the mapper.CartRAMBus interface.
func (cart *wicksteadDesign) PutRAM(_ int, idx int, data uint8) {
	cart.state.ram[idx] = data
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *wicksteadDesign) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))
	for b := 0; b < len(cart.banks); b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}
	return c
}

// rewindable state for the CBS cartridge.
type wicksteadState struct {
	// segments maps banks to addresses
	segments  [4]int
	pending   [4]int
	pendingCt int

	// WD cartridges have internal RAM very similar to the Atari Superchip
	ram []uint8
}

func newWicksteadState() *wicksteadState {
	const wicksteadRAMsize = 64

	return &wicksteadState{
		ram: make([]uint8, wicksteadRAMsize),
	}
}

// Snapshot implements the mapper.CartMapper interface.
func (s *wicksteadState) Snapshot() *wicksteadState {
	n := *s
	n.ram = make([]uint8, len(s.ram))
	copy(n.ram, s.ram)
	return &n
}
