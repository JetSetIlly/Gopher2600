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

package supercharger

import (
	"fmt"
	"path/filepath"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
)

// supercharger has 6k of RAM in total.
const numRAMBanks = 4
const bankSize = 2048

// The address in VCS RAM of the multiload byte. Tapes will not load unless the
// encoded multibyte in the stream is the same as the value in this address.
const MutliloadByteAddress = 0xfa

// tape defines the operations required by the $fff9 tape loader. With this
// interface, the Supercharger implementation supports both fast-loading
// from a Stella bin file, and "slow" loading from a sound file.
type tape interface {
	snapshot() tape
	plumb(*state, *environment.Environment)
	load() (uint8, error)
	step()
	end()
}

// Supercharger represents a supercharger cartridge.
type Supercharger struct {
	env *environment.Environment

	mappingID string

	bankSize int
	bios     []uint8

	// rewindable state
	state *state
}

// NewSupercharger is the preferred method of initialisation for the
// Supercharger type.
func NewSupercharger(env *environment.Environment, cartload cartridgeloader.Loader) (mapper.CartMapper, error) {
	cart := &Supercharger{
		env:       env,
		mappingID: "AR",
		bankSize:  2048,
		state:     newState(),
	}

	var err error

	// load bios and activate
	cart.bios, err = loadBIOS(env, filepath.Dir(cartload.Filename))
	if err != nil {
		return nil, fmt.Errorf("supercharger: %w", err)
	}

	// set up tape
	if cartload.IsSoundData {
		cart.state.tape, err = newSoundLoad(env, cartload)
	} else {
		cart.state.tape, err = newFastLoad(env, cart.state, cartload)
	}
	if err != nil {
		return nil, fmt.Errorf("supercharger: %w", err)
	}

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *Supercharger) MappedBanks() string {
	return cart.state.registers.MappedBanks()
}

// ID implements the mapper.CartMapper interface.
func (cart *Supercharger) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *Supercharger) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Supercharger) Plumb(env *environment.Environment) {
	cart.env = env
	cart.state.tape.plumb(cart.state, env)
}

// Reset implements the mapper.CartMapper interface.
func (cart *Supercharger) Reset() {
	for b := range cart.state.ram {
		for i := range cart.state.ram[b] {
			cart.state.ram[b][i] = uint8(cart.env.Random.NoRewind(0xff))
		}
	}

	cart.state.registers.WriteDelay = 0
	cart.state.registers.ROMpower = true
	cart.state.registers.RAMwrite = true

	cart.SetBank("AUTO")
}

// Access implements the mapper.CartMapper interface.
func (cart *Supercharger) Access(addr uint16, peek bool) (uint8, uint8, error) {
	// what bank to read. bank zero refers to the BIOS. bank 1 to 3 refer to
	// one of the RAM banks
	bank := cart.GetBank(addr).Number

	bios := false
	switch bank {
	case 0:
		bios = true
	default:
		// RAM banks are indexed from 0 to 2
		bank--
	}

	// tape load register has been read
	if addr == 0x0ff9 {
		// turn is loading state and call vcs hook if this is the first recent
		// read of the tape. we assume that the isLoading state will be
		// sustained until the BIOS is "touched" as described below
		if !cart.state.isLoading {
			cart.state.isLoading = true
		}

		// call load() whenever address is touched, although do not allow
		// it if RAMwrite is false
		if peek || !cart.state.registers.RAMwrite {
			return 0, 0, nil
		}

		v, err := cart.state.tape.load()
		if err != nil {
			err = fmt.Errorf("supercharger: %w", err)
		}
		return v, mapper.CartDrivenPins, err
	}

	// control register has been read. I've opted to return the value at the
	// address before the bank switch. I think this is correct but I'm not
	// sure.
	if addr == 0x0ff8 {
		b := cart.state.ram[bank][addr&0x07ff]
		if !peek {
			cart.state.registers.setConfigByte(cart.state.registers.Value)
			cart.state.registers.Delay = 0
		}
		return b, mapper.CartDrivenPins, nil
	}

	// note address to be used as the next value in the control register
	if !peek {
		if addr <= 0x00ff {
			if cart.state.registers.Delay == 0 {
				cart.state.registers.Value = uint8(addr)
				cart.state.registers.Delay = 6
			}
		}
	}

	if bios {
		if cart.state.registers.ROMpower {
			// send notification whenever BIOS address $fa1a (specifically) is
			// touched. note that this method means that the notification will
			// be sent whatever the context the address is read and not just
			// when the PC is at the address.
			if addr == 0x0a1a {
				// end tape is loading state
				cart.state.isLoading = false
				cart.state.tape.end()
			}

			return cart.bios[addr&0x07ff], mapper.CartDrivenPins, nil
		}

		return 0, 0, fmt.Errorf("supercharger: ROM is powered off")
	}

	if !peek && cart.state.registers.Delay == 1 {
		if cart.state.registers.RAMwrite {
			cart.state.ram[bank][addr&0x07ff] = cart.state.registers.Value
			cart.state.registers.LastWriteAddress = memorymap.OriginCart | addr
			cart.state.registers.LastWriteValue = cart.state.registers.Value
		}
	}

	return cart.state.ram[bank][addr&0x07ff], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *Supercharger) AccessVolatile(addr uint16, data uint8, _ bool) error {
	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *Supercharger) NumBanks() int {
	return numRAMBanks
}

// GetBank implements the mapper.CartMapper interface.
func (cart *Supercharger) GetBank(addr uint16) mapper.BankInfo {
	switch cart.state.registers.BankingMode {
	case 0:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 0, Name: "BIOS", IsRAM: false, IsSegmented: true, Segment: 1}
		}
		return mapper.BankInfo{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 1:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 0, Name: "BIOS", IsRAM: false, IsSegmented: true, Segment: 1}
		}
		return mapper.BankInfo{Number: 1, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}

	case 2:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 1, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}
		}
		return mapper.BankInfo{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 3:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}
		}
		return mapper.BankInfo{Number: 1, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 4:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 0, Name: "BIOS", IsRAM: false, IsSegmented: true, Segment: 1}
		}
		return mapper.BankInfo{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 5:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 0, Name: "BIOS", IsRAM: false, IsSegmented: true, Segment: 1}
		}
		return mapper.BankInfo{Number: 2, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 6:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 2, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}
		}
		return mapper.BankInfo{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 7:
		if addr >= 0x0800 {
			return mapper.BankInfo{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}
		}
		return mapper.BankInfo{Number: 2, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}
	}

	panic("supercharger: unknown banking method")
}

// SetBank implements the mapper.CartMapper interface.
func (cart *Supercharger) SetBank(bank string) error {
	if mapper.IsAutoBankSelection(bank) {
		cart.state.registers.BankingMode = 0
		return nil
	}

	// supercharger uses predfined banking modes. we can use the single bank
	// selection function for this

	b, err := mapper.SingleBankSelection(bank)
	if err != nil {
		return fmt.Errorf("%s: %w", cart.mappingID, err)
	}
	if b.IsRAM {
		return fmt.Errorf("%s: cartridge expects a pattern number between 0 and 7", cart.mappingID)
	}

	if b.Number > 7 {
		return fmt.Errorf("%s: invalid banking mode (%d)", cart.mappingID, b.Number)
	}

	cart.state.registers.BankingMode = b.Number

	return nil
}

// AccessPassive implements the mapper.CartMapper interface.
func (cart *Supercharger) AccessPassive(addr uint16, _ uint8) error {
	cart.state.registers.transitionCount(addr)
	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *Supercharger) Step(_ float32) {
	cart.state.tape.step()
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *Supercharger) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.state.ram)+1)

	c[0] = mapper.BankContent{Number: 0,
		Data: cart.bios,
		Origins: []uint16{
			memorymap.OriginCart,
			memorymap.OriginCart + uint16(cart.bankSize),
		},
	}

	for b := 0; b < len(cart.state.ram); b++ {
		c[b+1] = mapper.BankContent{Number: b + 1,
			Data: cart.state.ram[b],
			Origins: []uint16{
				memorymap.OriginCart,
				memorymap.OriginCart + uint16(cart.bankSize),
			},
		}
	}

	return c
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart *Supercharger) GetRAM() []mapper.CartRAM {
	r := make([]mapper.CartRAM, len(cart.state.ram))

	for i := 0; i < len(cart.state.ram); i++ {
		mapped := false
		origin := uint16(0x1000)

		// in the documentation and for presentation purporses, RAM banks are
		// counted from 1. when deciding if a bank is mapped or not, we'll use
		// this value rather than the i index; being consistent with the
		// documentation is clearer
		bank := i + 1

		switch cart.state.registers.BankingMode {
		case 0:
			mapped = bank == 3

		case 1:
			mapped = bank == 1

		case 2:
			mapped = bank == 1
			if mapped {
				origin = 0x1800
			} else {
				mapped = bank == 3
			}

		case 3:
			mapped = bank == 3
			if mapped {
				origin = 0x1800
			} else {
				mapped = bank == 1
			}

		case 4:
			mapped = bank == 3

		case 5:
			mapped = bank == 2

		case 6:
			mapped = bank == 2
			if mapped {
				origin = 0x1800
			} else {
				mapped = bank == 3
			}

		case 7:
			mapped = bank == 3
			if mapped {
				origin = 0x1800
			} else {
				mapped = bank == 2
			}
		}

		r[i] = mapper.CartRAM{
			Label:  fmt.Sprintf("2048k [%d]", bank),
			Origin: origin,
			Data:   make([]uint8, len(cart.state.ram[i])),
			Mapped: mapped,
		}
		copy(r[i].Data, cart.state.ram[i])
	}

	return r
}

// PutRAM implements the mapper.CartRAMBus interface.
func (cart *Supercharger) PutRAM(bank int, idx int, data uint8) {
	if bank < len(cart.state.ram) {
		cart.state.ram[bank][idx] = data
		return
	}
}

// Rewind implements the mapper.CartTapeBus interface
//
// Whether this does anything meaningful depends on the interal implementation
// of the 'tape' interface.
func (cart *Supercharger) Rewind() {
	if tape, ok := cart.state.tape.(mapper.CartTapeBus); ok {
		tape.Rewind()
	}
}

// SetTapeCounter implements the mapper.CartTapeBus interface
//
// Whether this does anything meaningful depends on the interal implementation
// of the 'tape' interface.
func (cart *Supercharger) SetTapeCounter(c int) {
	if tape, ok := cart.state.tape.(mapper.CartTapeBus); ok {
		tape.SetTapeCounter(c)
	}
}

// Fastload implements the mapper.CartSuperChargerFastLoad interface.
func (cart *Supercharger) Fastload(mc *cpu.CPU, ram *vcs.RAM, tmr *timer.Timer) error {
	if f, ok := cart.state.tape.(mapper.CartSuperChargerFastLoad); ok {
		return f.Fastload(mc, ram, tmr)
	}
	return nil
}

// GetTapeState implements the mapper.CartTapeBus interface
//
// Whether this does anything meaningful depends on the interal implementation
// of the 'tape' interface.
func (cart *Supercharger) GetTapeState() (bool, mapper.CartTapeState) {
	if tape, ok := cart.state.tape.(mapper.CartTapeBus); ok {
		return tape.GetTapeState()
	}
	return false, mapper.CartTapeState{}
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart *Supercharger) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff8: {Symbol: "CONFIG", Action: mapper.HotspotFunction},
		0x1ff9: {Symbol: "TAPE", Action: mapper.HotspotFunction},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *Supercharger) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return nil
}
