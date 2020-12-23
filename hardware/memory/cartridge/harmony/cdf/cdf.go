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

package cdf

import (
	"fmt"
	"math/rand"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony/arm7tdmi"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// cdf implements the cartMapper interface.
type cdf struct {
	mappingID string

	// cdf comes in several different versions
	version version

	// additional CPU - used by some ROMs
	arm *arm7tdmi.ARM

	// banks and the currently selected bank
	bankSize int
	banks    [][]byte

	// rewindable state
	state *State
}

// the sizes of these areas in a CDJF cartridge are fixed. the custom arm code
// (although it can expand into subsequent banks) and the 6507 program fit
// around these sizes.
const (
	driverSize = 2048 // 2k
	customSize = 2048 // 2k (may expand into subsequent banks)
)

// registers should be accessed via readDataFetcher() and updateDataFetcher().
// Actually reading the data in the data stream should be done by streamData().
//
// The following values can be used for convenience. The numbered datastreams
// can be accessed numerically as expected.
//
// The AMPLITUDE register must be accessed with version.amplitude because it
// can change depending on the CDF version being emulated.
const (
	DSCOMM = 32
	DSJMP  = 33
)

// NewCDF is the preferred method of initialisation for the harmony type.
func NewCDF(version byte, data []byte) (mapper.CartMapper, error) {
	cart := &cdf{
		mappingID: "CDF",
		bankSize:  4096,
		state:     newCDFstate(),
	}

	var err error
	cart.version, err = newVersion(version)
	if err != nil {
		return nil, curated.Errorf("CDF: %v", err)
	}

	// amount of data used for cartridges
	bankLen := len(data) - driverSize - customSize

	// size check
	if bankLen <= 0 || bankLen%cart.bankSize != 0 {
		return nil, curated.Errorf("CDF: wrong number of bytes in cartridge data")
	}

	// allocate enough banks
	cart.banks = make([][]uint8, bankLen/cart.bankSize)

	// partition data into banks
	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		offset += driverSize + customSize
		cart.banks[k] = data[offset : offset+cart.bankSize]
	}

	// initialise static memory
	cart.state.static = cart.newCDFstatic(data)

	// initialise ARM processor
	//
	// if bank0 has any ARM code then it will start at offset 0x08. first eight
	// bytes are the ARM header
	cart.arm = arm7tdmi.NewARM(cart.state.static, cart)

	return cart, nil
}

func (cart *cdf) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.mappingID, cart.version.description, cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart *cdf) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *cdf) Snapshot() mapper.CartSnapshot {
	return cart.state.Snapshot()
}

// Plumb implements the mapper.CartMapper interface.
func (cart *cdf) Plumb(s mapper.CartSnapshot) {
	cart.state = s.(*State)
	cart.arm.PlumbSharedMemory(cart.state.static)
}

// Reset implements the mapper.CartMapper interface.
func (cart *cdf) Reset(_ *rand.Rand) {
	bank := len(cart.banks) - 1
	if cart.version.submapping == "CDJF+" {
		bank = 0
	}
	cart.state.initialise(bank)
}

const (
	jmpAbsolute  = 0x4c
	ldaImmediate = 0xa9
)

// Read implements the mapper.CartMapper interface.
func (cart *cdf) Read(addr uint16, passive bool) (uint8, error) {
	if cart.bankswitch(addr, passive) {
		// always return zero on hotspot - unlike the Atari multi-bank carts for example
		return 0, nil
	}

	data := cart.banks[cart.state.bank][addr]

	if cart.state.registers.FastFetch && cart.state.fastJMP > 0 {
		// maybe surprisingly, a fastJMP bay bave be triggered erroneousy.
		//
		// how so? well, for example, a branch operator will cause a phantom read
		// before landing on the correct address. if the phantom read happens
		// to land on a byte of 0x4c and if it just so happens that the next
		// two bytes are value 0x00, then this FASTJMP branch will have been
		// triggered.
		//
		// believe it or not this actually happens in the Galagon NTSC demo
		// ROM. the BNE $f6fd instruction at $f72e (bank 6) will cause a
		// phantom read of address $f7fd. as luck would have it (or maybe not),
		// that address contains a sequence of $4c $00 $00. this causes the
		// FASTJMP to be initialised and eventually for a BRK instruction to be
		// returned at location $f6fd
		//
		// to mitigate this scenario, we take a note of what the operand
		// address *should* be and use that to discard false positives.
		if cart.state.fastJMP < 2 || (cart.state.fastJMP == 2 && cart.banks[cart.state.bank][addr-1] == jmpAbsolute) {
			// reduce jmp counter
			cart.state.fastJMP--

			// which register should we use
			reg := int(cart.banks[cart.state.bank][addr-1+uint16(cart.state.fastJMP)] + DSJMP)

			// get current address for the data stream
			jmp := cart.readDataFetcher(reg)
			data = cart.state.static.dataRAM[jmp>>cart.version.fetcherShift]
			jmp += 1 << cart.version.fetcherShift
			cart.updateDataFetcher(reg, jmp)

			return data, nil
		}
	}

	// any fastjmp preparation that wasn't serviced by the above branch must be
	// a false positive, by definition.
	cart.state.fastJMP = 0

	if cart.state.registers.FastFetch && cart.state.fastLDA {
		cart.state.fastLDA = false

		// data fetchers
		if data <= DSCOMM {
			return cart.streamData(int(data)), nil
		}

		// music fetchers
		if data == byte(cart.version.amplitudeRegister) {
			if cart.state.registers.SampleMode {
				addr := cart.readMusicFetcher(0)
				addr += cart.state.registers.MusicFetcher[0].Count >> 21

				// get sample from memory
				data = cart.state.static.read8bit(addr)

				// prevent excessive volume
				if cart.state.registers.MusicFetcher[0].Count&(1<<20) == 0 {
					data >>= 4
				}

				return data, nil
			}

			// data retrieval for non-SampleMode uses all three music fetchers
			data = 0
			for i := range cart.state.registers.MusicFetcher {
				m := cart.readMusicFetcher(i)
				m += (cart.state.registers.MusicFetcher[i].Count >> cart.state.registers.MusicFetcher[i].Waveform)
				data += cart.state.static.read8bit(m)
			}

			return data, nil
		}

		// if data is higher than AMPLITUDE then the 0xa9 we detected in the
		// previous Read() was just a normal value (maybe an LDA #immediate
		// opcode but not one intended for fast fetch)
	}

	// set lda flag if fast fetch mode is on and data returned is LDA #immediate
	cart.state.fastLDA = cart.state.registers.FastFetch && data == ldaImmediate

	// set jmp flag if fast fetch mode is on and data returned is JMP absolute
	if cart.state.registers.FastFetch && data == jmpAbsolute &&
		// only "jmp absolute" instructions with certain address operands are
		// treated as "FastJMPs". Generally, this address must be $0000 but in
		// the case of the CDFJ version an address of $0100 is also acceptable.
		cart.banks[cart.state.bank][addr+1]&cart.version.fastJMPmask == 0x00 && cart.banks[cart.state.bank][addr+2] == 0x00 {

		// JMP operator takes a 16bit operand so we'll count the number of
		// bytes we've read
		cart.state.fastJMP = 2
	}

	return data, nil
}

//

// Write implements the mapper.CartMapper interface.
func (cart *cdf) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if cart.bankswitch(addr, passive) {
		return nil
	}

	switch addr {
	case 0x0ff0:
		// DSWRITE

		// top 12 bits are significant
		v := cart.readDataFetcher(DSCOMM)

		// write data to ARM RAM
		cart.state.static.dataRAM[v>>cart.version.fetcherShift] = data

		// advance address value
		v += 1 << cart.version.fetcherShift

		// write adjusted address (making sure to put the bits in the top 12 bits)
		cart.updateDataFetcher(DSCOMM, v)

	case 0x0ff1:
		// DSPTR
		v := cart.readDataFetcher(DSCOMM) << 8
		v &= cart.version.fetcherMask

		// add new data to lower byte of dsptr value
		v |= (uint32(data) << cart.version.fetcherShift)

		// write dsptr to dscomm register
		cart.updateDataFetcher(DSCOMM, v)

	case 0x0ff2:
		// SETMODE
		cart.state.registers.FastFetch = data&0x0f != 0x0f
		cart.state.registers.SampleMode = data&0xf0 != 0xf0

		if !cart.state.registers.FastFetch {
			cart.state.fastLDA = false
			cart.state.fastJMP = 0
		}

	case 0x0ff3:
		fallthrough

	case 0x0ff4:
		// CALLFN
		switch data {
		case 0xfe:
			// generate interrupt to update AUDV0 while running ARM code
			fallthrough
		case 0xff:
			err := cart.arm.Run()
			if err != nil {
				return curated.Errorf("CDF: %v", err)
			}
		}
	}

	if poke {
		cart.banks[cart.state.bank][addr] = data
		return nil
	}

	return curated.Errorf("CDF: %v", curated.Errorf(bus.AddressError, addr))
}

// bankswitch on hotspot access.
func (cart *cdf) bankswitch(addr uint16, passive bool) bool {
	if addr >= 0x0ff4 && addr <= 0x0ffb {
		if passive {
			return true
		}

		if addr == 0x0ff4 {
			cart.state.bank = 6
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
		} else if addr == 0x0ffb {
			cart.state.bank = 6
		}

		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *cdf) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the mapper.CartMapper interface.
func (cart *cdf) GetBank(addr uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: cart.state.bank, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *cdf) Patch(offset int, data uint8) error {
	return curated.Errorf("CDF: patching unsupported")
}

// Listen implements the mapper.CartMapper interface.
func (cart *cdf) Listen(addr uint16, data uint8) {
}

// Step implements the mapper.CartMapper interface.
func (cart *cdf) Step() {
	// sample rate of 20KHz.
	//
	// Step() is called at a rate of 1.19Mhz. so:
	//
	// 1.19Mhz / 20KHz
	// = 59
	//
	// ie. we clock the music data fetchers once every 59 calls to Step()
	//
	// the 20Khz is the same as the DPC format (see mapper_dpc for commentary).

	cart.state.beats++
	if cart.state.beats%59 == 0 {
		cart.state.beats = 0
		cart.state.registers.MusicFetcher[0].Count += cart.state.registers.MusicFetcher[0].Freq
		cart.state.registers.MusicFetcher[1].Count += cart.state.registers.MusicFetcher[1].Freq
		cart.state.registers.MusicFetcher[2].Count += cart.state.registers.MusicFetcher[2].Freq
	}

	cart.arm.Step()
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *cdf) CopyBanks() []mapper.BankContent {
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
func (cart *cdf) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff5: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff6: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1ff7: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1ff8: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x1ffa: {Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
		0x1ffb: {Symbol: "BANK6", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *cdf) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff0: {Symbol: "DSWRITE", Action: mapper.HotspotRegister},
		0x1ff1: {Symbol: "DSPTR", Action: mapper.HotspotRegister},
		0x1ff2: {Symbol: "SETMODE", Action: mapper.HotspotRegister},
		0x1ff3: {Symbol: "CALLFN", Action: mapper.HotspotFunction},
		0x1ff6: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff7: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1ff8: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x1ffa: {Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x1ffb: {Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
	}
}

func (cart *cdf) updateDataFetcher(fetcher int, data uint32) {
	idx := cart.version.fetcherBase + (uint32(fetcher) * 4)
	cart.state.static.driverRAM[idx] = uint8(data)
	cart.state.static.driverRAM[idx+1] = uint8(data >> 8)
	cart.state.static.driverRAM[idx+2] = uint8(data >> 16)
	cart.state.static.driverRAM[idx+3] = uint8(data >> 24)
}

func (cart *cdf) readDataFetcher(reg int) uint32 {
	idx := cart.version.fetcherBase + (uint32(reg) * 4)
	return uint32(cart.state.static.driverRAM[idx]) |
		uint32(cart.state.static.driverRAM[idx+1])<<8 |
		uint32(cart.state.static.driverRAM[idx+2])<<16 |
		uint32(cart.state.static.driverRAM[idx+3])<<24
}

func (cart *cdf) readIncrement(reg int) uint32 {
	idx := cart.version.incrementBase + (uint32(reg) * 4)
	return uint32(cart.state.static.driverRAM[idx]) |
		uint32(cart.state.static.driverRAM[idx+1])<<8 |
		uint32(cart.state.static.driverRAM[idx+2])<<16 |
		uint32(cart.state.static.driverRAM[idx+3])<<24
}

func (cart *cdf) readMusicFetcher(reg int) uint32 {
	addr := cart.version.musicBase + (uint32(reg) * 4)
	return uint32(cart.state.static.driverRAM[addr]) |
		uint32(cart.state.static.driverRAM[addr+1])<<8 |
		uint32(cart.state.static.driverRAM[addr+2])<<16 |
		uint32(cart.state.static.driverRAM[addr+3])<<24
}

func (cart *cdf) streamData(reg int) uint8 {
	addr := cart.readDataFetcher(reg)
	inc := cart.readIncrement(reg)

	value := cart.state.static.dataRAM[addr>>cart.version.fetcherShift]
	addr += inc << cart.version.incrementShift
	cart.updateDataFetcher(reg, addr)

	return value
}

// ARMinterrupt implements the arm7tmdi.CatridgeHook interface.
func (cart *cdf) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm7tdmi.ARMinterruptReturn, error) {
	var r arm7tdmi.ARMinterruptReturn

	if cart.version.submapping == "CDF0" {
		switch addr {
		case 0x000006e2:
			// set note
			if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
				return r, curated.Errorf("music fetcher index (%d) too high ", val1)
			}
			cart.state.registers.MusicFetcher[val1].Freq = val2
		case 0x000006e6:
			// reset wave
			if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
				return r, curated.Errorf("music fetcher index (%d) too high ", val1)
			}
			cart.state.registers.MusicFetcher[val1].Count = 0
		case 0x000006ea:
			// get wave ptr
			if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
				return r, curated.Errorf("music fetcher index (%d) too high ", val1)
			}
			r.SaveValue = cart.state.registers.MusicFetcher[val1].Count
			r.SaveRegister = 2
			r.SaveResult = true
		case 0x000006ee:
			// set wave size
			if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
				return r, curated.Errorf("music fetcher index (%d) too high ", val1)
			}
			cart.state.registers.MusicFetcher[val1].Waveform = uint8(val2)
		default:
			return r, nil
		}

		r.InterruptServiced = true
		return r, nil
	}

	switch addr {
	case 0x00000752:
		// set note
		if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
			return r, curated.Errorf("music fetcher index (%d) too high ", val1)
		}
		cart.state.registers.MusicFetcher[val1].Freq = val2
	case 0x00000756:
		// reset wave
		if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
			return r, curated.Errorf("music fetcher index (%d) too high ", val1)
		}
		cart.state.registers.MusicFetcher[val1].Count = 0
	case 0x0000075a:
		// get wave ptr
		if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
			return r, curated.Errorf("music fetcher index (%d) too high ", val1)
		}
		r.SaveValue = cart.state.registers.MusicFetcher[val1].Count
		r.SaveRegister = 2
		r.SaveResult = true
	case 0x0000075e:
		// set wave size
		if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
			return r, curated.Errorf("music fetcher index (%d) too high ", val1)
		}
		cart.state.registers.MusicFetcher[val1].Waveform = uint8(val2)
	default:
		return r, nil
	}

	r.InterruptServiced = true
	return r, nil
}
