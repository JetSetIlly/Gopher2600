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
	"os"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources"
)

// FA2 is a variation of the CBS (FA mapper) and was originally intended to use
// the flash memory of the Harmony cartridge to store non-volatile data
type fa2 struct {
	env *environment.Environment

	mappingID string

	// fa2 cartridges have 3 banks of 4096 bytes
	bankSize int
	banks    [][]uint8

	// whether to support nvram reading and writing
	nvram bool

	// rewindable state
	state *fa2State
}

func newFA2(env *environment.Environment, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	data, err := io.ReadAll(loader)
	if err != nil {
		return nil, fmt.Errorf("FA2: %w", err)
	}

	cart := &fa2{
		env:       env,
		mappingID: "FA2",
		bankSize:  4096,
		state:     newFA2State(),
	}

	// FA2 can heave either 6 or 7 banks and maybe a section of ARM program data. in the
	// case of having ARM program data, the VCS data is offset by 1024 bytes
	var bankOffset int

	if len(data) >= 29696 {
		cart.banks = make([][]uint8, 7)
		cart.nvram = true
		bankOffset = 1024
	} else if len(data) == 28672 {
		cart.banks = make([][]uint8, 7)
	} else if len(data) == 24576 {
		cart.banks = make([][]uint8, 6)
	} else {
		return nil, fmt.Errorf("FA2: unsupported number of bytes in the cartridge data")
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		offset += bankOffset
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *fa2) MappedBanks() string {
	return fmt.Sprintf("Bank: %d", cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart *fa2) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *fa2) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *fa2) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *fa2) Reset() {
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
func (cart *fa2) Access(addr uint16, _ bool) (uint8, uint8, error) {
	if cart.hotspot(addr) {
		return 0, 0, nil
	}
	if addr <= 0x00ff {
		return 0, 0, nil
	}
	if addr >= 0x0100 && addr <= 0x01ff {
		return cart.state.ram[addr-0x100], mapper.CartDrivenPins, nil
	}
	return cart.banks[cart.state.bank][addr], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *fa2) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if cart.hotspot(addr) {
		return nil
	}
	if addr <= 0x00ff {
		cart.state.ram[addr] = data
		return nil
	}

	if poke {
		cart.banks[cart.state.bank][addr] = data
		return nil
	}

	return nil
}

// the resource path to the nvram files
const fa2_nvram = "fa2_nvram"

// hotspot is called from both Access() and AccessVolatile(). if the Harmony
// flash read/write time is implemented we'll need to take into account the
// double call on Access() - the memory system calls AccessVolatile() and
// Access() when a cartridge address is read
func (cart *fa2) hotspot(addr uint16) bool {
	if addr == 0x0ff4 && cart.nvram {
		switch cart.state.ram[0xff] {
		case 1:
			// read from nvram file

			// always indicate success by changing RAM value. a full emulation
			// of harmony flash memory would take into account the time it takes
			// for data to be read. but we're not doing that
			defer func() {
				cart.state.ram[0xff] = 0x00
			}()

			p, err := resources.JoinPath(fa2_nvram, cart.env.Loader.HashMD5)
			if err != nil {
				logger.Log(cart.env, "FA2", err)
				break // switch
			}

			st, err := os.Stat(p)
			if err != nil {
				logger.Log(cart.env, "FA2", err)
				break // switch
			}

			if st.Size() != 256 {
				logger.Logf(cart.env, "FA2", "%s is not 256 bytes in size", p)
				break // switch
			}

			f, err := os.Open(p)
			if err != nil {
				logger.Log(cart.env, "FA2", err)
				break // switch
			}

			n, err := f.Read(cart.state.ram)
			if err != nil {
				logger.Log(cart.env, "FA2", err)
			}
			if n != 256 {
				logger.Log(cart.env, "FA2", "expected 256 bytes in nvram file")
			}
			err = f.Close()
			if err != nil {
				logger.Log(cart.env, "FA2", err)
			}

		case 2:
			// write to nvram file

			// always indicate success by changing RAM value. a full emulation
			// of harmony flash memory would take into account the time it takes
			// for data to be written. but we're not doing that
			defer func() {
				cart.state.ram[0xff] = 0x00
			}()

			p, err := resources.JoinPath(fa2_nvram, cart.env.Loader.HashMD5)
			if err != nil {
				logger.Log(cart.env, "FA2", err)
				break // switch
			}

			f, err := os.Create(p)
			if err != nil {
				logger.Log(cart.env, "FA2", err)
				break // switch
			}

			n, err := f.Write(cart.state.ram)
			if err != nil {
				logger.Log(cart.env, "FA2", err)
			}
			if n != 256 {
				logger.Log(cart.env, "FA2", "failed to write 256 bytes to nvram file")
			}
			err = f.Close()
			if err != nil {
				logger.Log(cart.env, "FA2", err)
			}
		}
	} else if addr == 0x0ff5 {
		cart.state.bank = 0
	} else if addr == 0x0ff6 {
		cart.state.bank = 1
	} else if addr == 0x0ff7 {
		cart.state.bank = 2
	} else if addr == 0x0ff8 {
		cart.state.bank = 3
	} else if addr == 0x0ff9 {
		cart.state.bank = 4
	} else if addr == 0x0ffa {
		cart.state.bank = 5
	} else if addr == 0x0ffb && len(cart.banks) > 6 {
		cart.state.bank = 6
	} else {
		return false
	}

	return true
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *fa2) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the mapper.CartMapper interface.
func (cart *fa2) GetBank(addr uint16) mapper.BankInfo {
	// fa2 cartridges are like atari cartridges in that the entire address
	// space points to the selected bank
	return mapper.BankInfo{Number: cart.state.bank, IsRAM: addr <= 0x00ff}
}

// SetBank implements the mapper.CartMapper interface.
func (cart *fa2) SetBank(bank string) error {
	if mapper.IsAutoBankSelection(bank) {
		cart.state.bank = 0
		return nil
	}

	b, err := mapper.SingleBankSelection(bank)
	if err != nil {
		return fmt.Errorf("%s: %v", cart.mappingID, err)
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
func (cart *fa2) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return fmt.Errorf("FA2: patch offset too high (%d)", offset)
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *fa2) AccessPassive(addr uint16, data uint8) error {
	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *fa2) Step(_ float32) {
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart *fa2) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, 1)
	r[0] = mapper.CartRAM{
		Label:  "FA2",
		Origin: 0x1100,
		Data:   make([]uint8, len(cart.state.ram)),
		Mapped: true,
	}
	copy(r[0].Data, cart.state.ram)
	return r
}

// PutRAM implements the mapper.CartRAMBus interface.
func (cart *fa2) PutRAM(_ int, idx int, data uint8) {
	cart.state.ram[idx] = data
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *fa2) CopyBanks() []mapper.BankContent {
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
func (cart *fa2) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	m := map[uint16]mapper.CartHotspotInfo{
		0x1ff5: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff6: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1ff7: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1ff8: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x1ffa: {Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
	}
	if cart.nvram {
		m[0x1ff4] = mapper.CartHotspotInfo{Symbol: "NVRAM", Action: mapper.HotspotFunction}
	}
	if len(cart.banks) > 6 {
		m[0x1ffb] = mapper.CartHotspotInfo{Symbol: "BANK6", Action: mapper.HotspotBankSwitch}
	}
	return m
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *fa2) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}

// rewindable state for the FA2 cartridge.
type fa2State struct {
	bank int
	ram  []uint8
}

func newFA2State() *fa2State {
	const fa2RAMsize = 256

	return &fa2State{
		ram: make([]uint8, fa2RAMsize),
	}
}

// Snapshot implements the mapper.CartMapper interface.
func (s *fa2State) Snapshot() *fa2State {
	n := *s
	n.ram = make([]uint8, len(s.ram))
	copy(n.ram, s.ram)
	return &n
}
