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
	"io"
	"os"
	"path/filepath"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper/banking"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/logger"
)

const (
	// supercharger has 6k of RAM in total.
	numRAMBanks = 4
	bankSize    = 2048

	// The address in VCS RAM of the multiload byte. Tapes will not load unless the
	// encoded multibyte in the stream is the same as the value in this address.
	MutliloadByteAddr = uint16(0x00fa)

	// RAM location where the JMP address instruction is placed for the bootstrap
	jmpAddrLo = uint16(0x00fe)
	jmpAddrHi = uint16(0x00ff)

	// location of config byte at moment of bootstrap
	configByteAddr = uint16(0x0080)
)

// tape defines the operations required by the $fff9 tape loader. With this
// interface, the Supercharger implementation supports both fast-loading
// from a Stella bin file, and "slow" loading from a sound file.
type tape interface {
	snapshot() tape
	plumb(*environment.Environment)
	load() (uint8, error)
	step()
	end()
	romdump(io.Writer) error
	bootstrap(*state, *cpu.CPU, *vcs.RAM, *timer.Timer, *tia.TIA) error
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
func NewSupercharger(env *environment.Environment) (mapper.CartMapper, error) {
	cart := &Supercharger{
		env:       env,
		mappingID: "AR",
		bankSize:  2048,
		state:     newState(),
	}

	var err error

	// load bios and activate
	if env.Loader.IsSoundData {
		cart.bios, err = loadBIOS(env, filepath.Dir(env.Loader.Filename))
		if err != nil {
			return nil, fmt.Errorf("supercharger: %w", err)
		}
	} else {
		cart.bios = fastloadOnlyBIOS()
	}

	// set up tape
	if env.Loader.IsSoundData {
		cart.state.tape, err = newSoundLoad(env)
	} else {
		cart.state.tape, err = newFastLoad(env)
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
	cart.state.tape.plumb(env)
}

// Reset implements the mapper.CartMapper interface.
func (cart *Supercharger) Reset() error {
	for b := range cart.state.ram {
		for i := range cart.state.ram[b] {
			cart.state.ram[b][i] = uint8(cart.env.Random.Intn(0xff))
		}
	}

	cart.state.registers.WriteDelay = 0
	cart.state.registers.ROMpower = true
	cart.state.registers.RAMwrite = true

	cart.SetBank("AUTO")

	return nil
}

// Access implements the mapper.CartMapper interface.
func (cart *Supercharger) access(addr uint16, peek bool) (uint8, uint8, error) {
	// what bank to read. bank zero refers to the BIOS. bank 1 to 3 refer to
	// one of the RAM banks
	bank := cart.GetBank(addr).Number

	// use bios data rather than ram data if bank number is zero
	bios := bank == 0

	// RAM banks are indexed from 0 to 2. this also has the effect of making the bank value to be
	// unsuitable for use as an index number in the case of the address pointing to the bios. that's
	// okay because it will cause a program panic, which is what we want
	bank--

	// always record access addresss for later comparison
	defer func() {
		cart.state.recentAddresses[1] = cart.state.recentAddresses[0]
		cart.state.recentAddresses[0] = addr
	}()

	// tape load register has been read
	switch addr {
	case 0x0ff9:
		// for a long time, we chose to do nothing with the tape load request unless 'RAM Write' was
		// enabled. however this isn't correct because it's possible for non-BIOS code to want to
		// read data from the tape

		// it's possible for the tape register to be triggered by a phantom access. this isn't a
		// problem for soundloud but it is a problem for fastload because a call to the tape.load()
		// function will immediately send the FastLoad notificatoin resulting in a call to the
		// bootstrap() function
		//
		// this happens with the original starpath game 'suicide mission'. the following filter
		// prevents calls to the tape.load() function while preserving the ability of ROMs to
		// intentionally load data from a tape
		if (cart.state.recentAddresses[0] == 0x0ff9 || cart.state.recentAddresses[0] == 0x0ff8) &&
			cart.state.recentAddresses[1] == 0x0ff8 {

			return 0x00, mapper.CartDrivenPins, nil
		}

		v, err := cart.state.tape.load()
		if err != nil {
			err = fmt.Errorf("supercharger: %w", err)
		}

		return v, mapper.CartDrivenPins, err

	case 0x0ff8:
		if !peek {
			cart.state.registers.setConfigByte(cart.state.registers.Value)
			cart.state.registers.Delay = 0
		}
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
			if addr == loadEndedAddress {
				cart.state.tape.end()
			}
			return cart.bios[addr&0x07ff], mapper.CartDrivenPins, nil
		}

		return 0, mapper.CartDrivenPins, fmt.Errorf("supercharger: ROM is powered off")
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

func (cart *Supercharger) Access(addr uint16, peek bool) (uint8, uint8, error) {
	return cart.access(addr, peek)
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *Supercharger) AccessVolatile(addr uint16, data uint8, poke bool) error {
	if poke {
		bank := cart.GetBank(addr).Number
		switch bank {
		case 0:
			cart.bios[addr&0x07ff] = data
		default:
			cart.state.ram[bank-1][addr&0x07ff] = data
		}
		return nil
	}
	_, _, err := cart.access(addr, false)
	return err
}

func (cart *Supercharger) biosAvailable() bool {
	return cart.state.registers.BankingMode == 0 ||
		cart.state.registers.BankingMode == 1 ||
		cart.state.registers.BankingMode == 4 ||
		cart.state.registers.BankingMode == 5
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *Supercharger) NumBanks() int {
	return numRAMBanks
}

// GetBank implements the mapper.CartMapper interface.
func (cart *Supercharger) GetBank(addr uint16) banking.Information {
	switch cart.state.registers.BankingMode {
	case 0:
		if addr >= 0x0800 {
			return banking.Information{Number: 0, Name: "BIOS", IsRAM: false, IsSegmented: true, Segment: 1}
		}
		return banking.Information{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 1:
		if addr >= 0x0800 {
			return banking.Information{Number: 0, Name: "BIOS", IsRAM: false, IsSegmented: true, Segment: 1}
		}
		return banking.Information{Number: 1, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}

	case 2:
		if addr >= 0x0800 {
			return banking.Information{Number: 1, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}
		}
		return banking.Information{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 3:
		if addr >= 0x0800 {
			return banking.Information{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}
		}
		return banking.Information{Number: 1, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 4:
		if addr >= 0x0800 {
			return banking.Information{Number: 0, Name: "BIOS", IsRAM: false, IsSegmented: true, Segment: 1}
		}
		return banking.Information{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 5:
		if addr >= 0x0800 {
			return banking.Information{Number: 0, Name: "BIOS", IsRAM: false, IsSegmented: true, Segment: 1}
		}
		return banking.Information{Number: 2, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 6:
		if addr >= 0x0800 {
			return banking.Information{Number: 2, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}
		}
		return banking.Information{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}

	case 7:
		if addr >= 0x0800 {
			return banking.Information{Number: 3, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 1}
		}
		return banking.Information{Number: 2, IsRAM: cart.state.registers.RAMwrite, IsSegmented: true, Segment: 0}
	}

	panic("supercharger: unknown banking method")
}

// SetBank implements the mapper.CartMapper interface.
func (cart *Supercharger) SetBank(bank string) error {
	if banking.IsAutoSelection(bank) {
		cart.state.registers.BankingMode = 0
		return nil
	}

	// supercharger uses predfined banking modes. we can use the single bank
	// selection function for this

	b, err := banking.SingleSelection(bank)
	if err != nil {
		return fmt.Errorf("supercharger: %w", err)
	}
	if b.IsRAM {
		return fmt.Errorf("supercharger: cartridge expects a pattern number between 0 and 7")
	}

	if b.Number > 7 {
		return fmt.Errorf("supercharger: invalid banking mode (%d)", b.Number)
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
func (cart *Supercharger) CopyBanks() []banking.Content {
	c := make([]banking.Content, len(cart.state.ram)+1)

	c[0] = banking.Content{Number: 0,
		Data: cart.bios,
		Origins: []uint16{
			memorymap.OriginCart,
			memorymap.OriginCart + uint16(cart.bankSize),
		},
	}

	for b := range len(cart.state.ram) {
		c[b+1] = banking.Content{Number: b + 1,
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

	for i := range len(cart.state.ram) {
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

// Bootstrap implements the mapper.CartSuperchargerBootstrap interface.
func (cart *Supercharger) Bootstrap(mc *cpu.CPU, ram *vcs.RAM, tmr *timer.Timer, tia *tia.TIA) error {
	return cart.state.tape.bootstrap(cart.state, mc, ram, tmr, tia)
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

// Hotspots implements the mapper.CartHotspotsBus interface.
func (cart *Supercharger) Hotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff8: {Symbol: "CONFIG", Action: mapper.HotspotFunction},
		0x1ff9: {Symbol: "TAPE", Action: mapper.HotspotFunction},
	}
}

func (cart *Supercharger) ROMDump(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("supercharger: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf(cart.env, "supercharger", "%v", err)
		}
	}()

	return cart.state.tape.romdump(f)
}
