// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be usdful,
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

type df struct {
	env *environment.Environment

	mappingID string

	// df cartridges have 3 banks of 4096 bytes
	bankSize int
	banks    [][]uint8

	// rewindable state
	state *dfState
}

// example ROM: penult RPG.
func newDF(env *environment.Environment, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	data, err := io.ReadAll(loader)
	if err != nil {
		return nil, fmt.Errorf("DF: %w", err)
	}

	cart := &df{
		env:       env,
		mappingID: "DF",
		bankSize:  4096,
		state:     newDfState(),
	}

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, fmt.Errorf("DF: wrong number of bytes in the cartridge data")
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *df) MappedBanks() string {
	return fmt.Sprintf("Bank: %d", cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart *df) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *df) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *df) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *df) Reset() {
	for i := range cart.state.ram {
		if cart.env.Prefs.RandomState.Get().(bool) {
			cart.state.ram[i] = uint8(cart.env.Random.NoRewind(0xff))
		} else {
			cart.state.ram[i] = 0
		}
	}

	cart.SetBank("AUTO")
}

// Access implements the mapper.CartMapper interface.
func (cart *df) Access(addr uint16, peek bool) (uint8, uint8, error) {
	if addr <= 0x007f {
		return 0, 0, nil
	}
	if addr >= 0x0080 && addr <= 0x00ff {
		return cart.state.ram[addr-0x80], mapper.CartDrivenPins, nil
	}

	if !peek {
		cart.bankswitch(addr)
	}

	return cart.banks[cart.state.bank][addr], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *df) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if !poke {
		if cart.bankswitch(addr) {
			return nil
		}
	}

	if addr <= 0x007f {
		cart.state.ram[addr] = data
		return nil
	}

	if poke {
		cart.banks[cart.state.bank][addr] = data
		return nil
	}

	return nil
}

// bankswitch on hotspot access.
func (cart *df) bankswitch(addr uint16) bool {
	if addr >= 0x0fc0 && addr <= 0xfdf {
		// looking at this switch, I'm now thinking hotspots could be done
		// programmatically. for now though, we'll keep it like this.
		if addr == 0x0fc0 {
			cart.state.bank = 0
		} else if addr == 0x0fc1 {
			cart.state.bank = 1
		} else if addr == 0x0fc2 {
			cart.state.bank = 2
		} else if addr == 0x0fc3 {
			cart.state.bank = 3
		} else if addr == 0x0fc4 {
			cart.state.bank = 4
		} else if addr == 0x0fc5 {
			cart.state.bank = 5
		} else if addr == 0x0fc6 {
			cart.state.bank = 6
		} else if addr == 0x0fc7 {
			cart.state.bank = 7
		} else if addr == 0x0fc8 {
			cart.state.bank = 8
		} else if addr == 0x0fc9 {
			cart.state.bank = 9
		} else if addr == 0x0fca {
			cart.state.bank = 10
		} else if addr == 0x0fcb {
			cart.state.bank = 11
		} else if addr == 0x0fcc {
			cart.state.bank = 12
		} else if addr == 0x0fcd {
			cart.state.bank = 13
		} else if addr == 0x0fce {
			cart.state.bank = 14
		} else if addr == 0x0fcf {
			cart.state.bank = 15
		} else if addr == 0x0fd0 {
			cart.state.bank = 16
		} else if addr == 0x0fd1 {
			cart.state.bank = 17
		} else if addr == 0x0fd2 {
			cart.state.bank = 18
		} else if addr == 0x0fd3 {
			cart.state.bank = 19
		} else if addr == 0x0fd4 {
			cart.state.bank = 20
		} else if addr == 0x0fd5 {
			cart.state.bank = 21
		} else if addr == 0x0fd6 {
			cart.state.bank = 22
		} else if addr == 0x0fd7 {
			cart.state.bank = 23
		} else if addr == 0x0fd8 {
			cart.state.bank = 24
		} else if addr == 0x0fd9 {
			cart.state.bank = 25
		} else if addr == 0x0fda {
			cart.state.bank = 26
		} else if addr == 0x0fdb {
			cart.state.bank = 27
		} else if addr == 0x0fdc {
			cart.state.bank = 28
		} else if addr == 0x0fdd {
			cart.state.bank = 29
		} else if addr == 0x0fde {
			cart.state.bank = 30
		} else if addr == 0x0fdf {
			cart.state.bank = 31
		}
		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *df) NumBanks() int {
	return 32
}

// GetBank implements the mapper.CartMapper interface.
func (cart *df) GetBank(addr uint16) mapper.BankInfo {
	// df cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return mapper.BankInfo{Number: cart.state.bank, IsRAM: addr <= 0x00ff}
}

// SetBank implements the mapper.CartMapper interface.
func (cart *df) SetBank(bank string) error {
	if mapper.IsAutoBankSelection(bank) {
		cart.state.bank = 15
		return nil
	}

	b, err := mapper.SingleBankSelection(bank)
	if err != nil {
		return fmt.Errorf("%s: %w", cart.mappingID, err)
	}

	if b.Number >= len(cart.banks) {
		return fmt.Errorf("%s: cartridge does not have bank '%d'", cart.mappingID, b.Number)
	}
	if b.IsRAM {
		return fmt.Errorf("%s: cartridge does not have bankable RAM", cart.mappingID)
	}

	cart.state.bank = b.Number

	return nil
}

// Patch implements the mapper.CartPatchable interface
func (cart *df) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return fmt.Errorf("DF: patch offset too high (%d)", offset)
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *df) AccessPassive(addr uint16, data uint8) error {
	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *df) Step(_ float32) {
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart *df) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, 1)
	r[0] = mapper.CartRAM{
		Label:  "DF+RAM",
		Origin: 0x1080,
		Data:   make([]uint8, len(cart.state.ram)),
		Mapped: true,
	}
	copy(r[0].Data, cart.state.ram)
	return r
}

// PutRAM implements the mapper.CartRAMBus interface.
func (cart *df) PutRAM(_ int, idx int, data uint8) {
	cart.state.ram[idx] = data
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *df) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))
	for b := 0; b < len(cart.banks); b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}
	return c
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart *df) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1fc0: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1fc1: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1fc2: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1fc3: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x1fc4: {Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x1fc5: {Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
		0x1fc6: {Symbol: "BANK6", Action: mapper.HotspotBankSwitch},
		0x1fc7: {Symbol: "BANK7", Action: mapper.HotspotBankSwitch},
		0x1fc8: {Symbol: "BANK8", Action: mapper.HotspotBankSwitch},
		0x1fc9: {Symbol: "BANK9", Action: mapper.HotspotBankSwitch},
		0x1fca: {Symbol: "BANK10", Action: mapper.HotspotBankSwitch},
		0x1fcb: {Symbol: "BANK11", Action: mapper.HotspotBankSwitch},
		0x1fcc: {Symbol: "BANK12", Action: mapper.HotspotBankSwitch},
		0x1fcd: {Symbol: "BANK13", Action: mapper.HotspotBankSwitch},
		0x1fce: {Symbol: "BANK14", Action: mapper.HotspotBankSwitch},
		0x1fcf: {Symbol: "BANK15", Action: mapper.HotspotBankSwitch},
		0x1fd0: {Symbol: "BANK16", Action: mapper.HotspotBankSwitch},
		0x1fd1: {Symbol: "BANK17", Action: mapper.HotspotBankSwitch},
		0x1fd2: {Symbol: "BANK18", Action: mapper.HotspotBankSwitch},
		0x1fd3: {Symbol: "BANK19", Action: mapper.HotspotBankSwitch},
		0x1fd4: {Symbol: "BANK20", Action: mapper.HotspotBankSwitch},
		0x1fd5: {Symbol: "BANK21", Action: mapper.HotspotBankSwitch},
		0x1fd6: {Symbol: "BANK22", Action: mapper.HotspotBankSwitch},
		0x1fd7: {Symbol: "BANK23", Action: mapper.HotspotBankSwitch},
		0x1fd8: {Symbol: "BANK24", Action: mapper.HotspotBankSwitch},
		0x1fd9: {Symbol: "BANK25", Action: mapper.HotspotBankSwitch},
		0x1fda: {Symbol: "BANK26", Action: mapper.HotspotBankSwitch},
		0x1fdb: {Symbol: "BANK27", Action: mapper.HotspotBankSwitch},
		0x1fdc: {Symbol: "BANK28", Action: mapper.HotspotBankSwitch},
		0x1fdd: {Symbol: "BANK29", Action: mapper.HotspotBankSwitch},
		0x1fde: {Symbol: "BANK30", Action: mapper.HotspotBankSwitch},
		0x1fdf: {Symbol: "BANK31", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *df) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}

// rewindable state for the cartridge type.
type dfState struct {
	// identifies the currently selected bank
	bank int

	// df cartridges always have a RAM area
	ram []uint8
}

func newDfState() *dfState {
	const dfRAMsize = 256

	return &dfState{
		ram: make([]uint8, dfRAMsize),
	}
}

func (s *dfState) Snapshot() *dfState {
	n := *s
	n.ram = make([]uint8, len(s.ram))
	copy(n.ram, s.ram)
	return &n
}
