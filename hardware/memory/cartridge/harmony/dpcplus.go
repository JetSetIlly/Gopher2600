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

package harmony

import (
	"fmt"
	"math/rand"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// dpcPlus implements the cartMapper interface.
//
// https://atariage.com/forums/topic/163495-harmony-dpc-programming
type dpcPlus struct {
	mappingID   string
	description string

	// banks and the currently selected bank
	bankSize int
	banks    [][]byte
	bank     int

	// static area of the cartridge. accessible outside of the cartridge
	// through GetStatic() and PutStatic()
	static DPCplusStatic

	// rewindable state
	state *dpcPlusState

	// patch help. offsets in the original data file for the different areas
	// in the cartridge
	//
	// we only do this because of the complexity of the dpcPlus file and only
	// for the purposes of the Patch() function. we don't bother with anything
	// like this for the simpler cartridge formats
	banksOffset int
	dataOffset  int
	freqOffset  int
	fileSize    int
}

// NewDPCplus is the preferred method of initialisation for the harmony type.
func NewDPCplus(data []byte) (mapper.CartMapper, error) {
	const armSize = 3072
	const dataSize = 4096
	const freqSize = 1024

	cart := &dpcPlus{
		mappingID:   "DPC+",
		description: "harmony",
		bankSize:    4096,
		state:       newDPCPlusState(),
	}

	// amount of data used for cartridges
	bankLen := len(data) - dataSize - armSize - freqSize

	// size check
	if bankLen <= 0 || bankLen%cart.bankSize != 0 {
		return nil, curated.Errorf("DPC+: %v", fmt.Errorf("%s: wrong number of bytes in cartridge data", cart.mappingID))
	}

	// partition
	cart.static.Arm = data[:armSize]

	// allocate enough banks
	cart.banks = make([][]uint8, bankLen/cart.bankSize)

	// partition data into banks
	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		offset += armSize
		cart.banks[k] = data[offset : offset+cart.bankSize]
	}

	// gfx and frequency table at end of file
	dataOffset := armSize + (cart.bankSize * cart.NumBanks())
	cart.static.Data = data[dataOffset : dataOffset+dataSize]
	cart.static.Freq = data[dataOffset+dataSize:]

	// patch offsets
	cart.banksOffset = armSize
	cart.dataOffset = dataOffset
	cart.freqOffset = dataOffset + dataSize
	cart.fileSize = len(data)

	return cart, nil
}

func (cart dpcPlus) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.mappingID, cart.description, cart.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart dpcPlus) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *dpcPlus) Snapshot() mapper.CartSnapshot {
	return cart.state.Snapshot()
}

// Plumb implements the mapper.CartMapper interface.
func (cart *dpcPlus) Plumb(s mapper.CartSnapshot) {
	cart.state = s.(*dpcPlusState)
}

// Reset implements the mapper.CartMapper interface.
func (cart *dpcPlus) Reset(randSrc *rand.Rand) {
	cart.state.registers.reset(randSrc)
	cart.bank = len(cart.banks) - 1
}

// Read implements the mapper.CartMapper interface.
func (cart *dpcPlus) Read(addr uint16, passive bool) (uint8, error) {
	if cart.bankswitch(addr, passive) {
		// always return zero on hotspot - unlike the Atari multi-bank carts for example
		return 0, nil
	}

	var data uint8

	// if address is above register space then we only need to check for bank
	// switching before returning data at the quoted address
	if addr > 0x007f {
		data = cart.banks[cart.bank][addr]

		// if FastFetch mode is on and the preceding data value was 0xa9 (the
		// opcode for LDA <immediate>) then the data we've just read this cycle
		// should be interpreted as an address to read from. we can do this by
		// recursing into the Read() function (there is no worry about deep
		// recursions because we reset the lda flag before recursing and the
		// lda flag being set is a prerequisite for the recursion to take
		// place)
		if cart.state.registers.FastFetch && cart.state.lda && data < 0x28 {
			cart.state.lda = false
			return cart.Read(uint16(data), passive)
		}

		cart.state.lda = cart.state.registers.FastFetch && data == 0xa9
		return data, nil
	}

	if addr > 0x0027 {
		return 0, curated.Errorf("DPC+: %v", curated.Errorf(bus.AddressError, addr))
	}

	switch addr {
	// random number generator
	case 0x00:
		cart.state.registers.RNG.next()
		data = uint8(cart.state.registers.RNG.Value)
	case 0x01:
		cart.state.registers.RNG.prev()
		data = uint8(cart.state.registers.RNG.Value)
	case 0x02:
		data = uint8(cart.state.registers.RNG.Value >> 8)
	case 0x03:
		data = uint8(cart.state.registers.RNG.Value >> 16)
	case 0x04:
		data = uint8(cart.state.registers.RNG.Value >> 24)

	// music fetcher
	case 0x05:
		data = cart.static.Data[(cart.state.registers.MusicFetcher[0].Waveform<<5)+(cart.state.registers.MusicFetcher[0].Count>>27)]
		data += cart.static.Data[(cart.state.registers.MusicFetcher[1].Waveform<<5)+(cart.state.registers.MusicFetcher[1].Count>>27)]
		data += cart.static.Data[(cart.state.registers.MusicFetcher[2].Waveform<<5)+(cart.state.registers.MusicFetcher[2].Count>>27)]

	// reserved
	case 0x06:
	case 0x07:

	// data fetcher
	case 0x08:
		fallthrough
	case 0x09:
		fallthrough
	case 0x0a:
		fallthrough
	case 0x0b:
		fallthrough
	case 0x0c:
		fallthrough
	case 0x0d:
		fallthrough
	case 0x0e:
		fallthrough
	case 0x0f:
		f := addr & 0x0007
		dataAddr := uint16(cart.state.registers.Fetcher[f].Hi)<<8 | uint16(cart.state.registers.Fetcher[f].Low)
		dataAddr &= 0x0fff
		data = cart.static.Data[dataAddr]
		cart.state.registers.Fetcher[f].inc()

	// data fetcher (windowed)
	case 0x10:
		fallthrough
	case 0x11:
		fallthrough
	case 0x12:
		fallthrough
	case 0x13:
		fallthrough
	case 0x14:
		fallthrough
	case 0x15:
		fallthrough
	case 0x16:
		fallthrough
	case 0x17:
		f := addr & 0x0007
		dataAddr := uint16(cart.state.registers.Fetcher[f].Hi)<<8 | uint16(cart.state.registers.Fetcher[f].Low)
		dataAddr &= 0x0fff
		if cart.state.registers.Fetcher[f].isWindow() {
			data = cart.static.Data[dataAddr]
		}
		cart.state.registers.Fetcher[f].inc()

	// fractional data fetcher
	case 0x18:
		fallthrough
	case 0x19:
		fallthrough
	case 0x1a:
		fallthrough
	case 0x1b:
		fallthrough
	case 0x1c:
		fallthrough
	case 0x1d:
		fallthrough
	case 0x1e:
		fallthrough
	case 0x1f:
		f := addr & 0x0007
		dataAddr := uint16(cart.state.registers.FracFetcher[f].Hi)<<8 | uint16(cart.state.registers.FracFetcher[f].Low)
		dataAddr &= 0x0fff
		data = cart.static.Data[dataAddr]
		cart.state.registers.FracFetcher[f].inc()

	// data fetcher window flag
	case 0x20:
		fallthrough
	case 0x21:
		fallthrough
	case 0x22:
		fallthrough
	case 0x23:
		f := addr & 0x0007
		if cart.state.registers.Fetcher[f].isWindow() {
			data = 0xff
		}

	// reserved
	case 0x24:
	case 0x25:
	case 0x26:
	case 0x27:
	}

	return data, nil
}

// Write implements the mapper.CartMapper interface.
func (cart *dpcPlus) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if cart.bankswitch(addr, passive) {
		return nil
	}

	if addr < 0x0028 || addr > 0x007f {
		return curated.Errorf("DPC+: %v", curated.Errorf(bus.AddressError, addr))
	}

	switch addr {
	// fractional data fetcher, low
	case 0x28:
		fallthrough
	case 0x29:
		fallthrough
	case 0x2a:
		fallthrough
	case 0x2b:
		fallthrough
	case 0x2c:
		fallthrough
	case 0x2d:
		fallthrough
	case 0x2e:
		fallthrough
	case 0x2f:
		f := addr & 0x0007
		cart.state.registers.FracFetcher[f].Low = data
		cart.state.registers.FracFetcher[f].Count = 0

	// fractional data fetcher, high
	case 0x30:
		fallthrough
	case 0x31:
		fallthrough
	case 0x32:
		fallthrough
	case 0x33:
		fallthrough
	case 0x34:
		fallthrough
	case 0x35:
		fallthrough
	case 0x36:
		fallthrough
	case 0x37:
		f := addr & 0x0007
		cart.state.registers.FracFetcher[f].Hi = data
		cart.state.registers.FracFetcher[f].Count = 0

	// fractional data fetcher, incrememnt
	case 0x38:
		fallthrough
	case 0x39:
		fallthrough
	case 0x3a:
		fallthrough
	case 0x3b:
		fallthrough
	case 0x3c:
		fallthrough
	case 0x3d:
		fallthrough
	case 0x3e:
		fallthrough
	case 0x3f:
		f := addr & 0x0007
		cart.state.registers.FracFetcher[f].Increment = data
		cart.state.registers.FracFetcher[f].Count = 0

	// data fetcher, window top
	case 0x40:
		fallthrough
	case 0x41:
		fallthrough
	case 0x42:
		fallthrough
	case 0x43:
		fallthrough
	case 0x44:
		fallthrough
	case 0x45:
		fallthrough
	case 0x46:
		fallthrough
	case 0x47:
		f := addr & 0x0007
		cart.state.registers.Fetcher[f].Top = data

	// data fetcher, window bottom
	case 0x48:
		fallthrough
	case 0x49:
		fallthrough
	case 0x4a:
		fallthrough
	case 0x4b:
		fallthrough
	case 0x4c:
		fallthrough
	case 0x4d:
		fallthrough
	case 0x4e:
		fallthrough
	case 0x4f:
		f := addr & 0x0007
		cart.state.registers.Fetcher[f].Bottom = data

	// data fetcher, low pointer
	case 0x50:
		fallthrough
	case 0x51:
		fallthrough
	case 0x52:
		fallthrough
	case 0x53:
		fallthrough
	case 0x54:
		fallthrough
	case 0x55:
		fallthrough
	case 0x56:
		fallthrough
	case 0x57:
		f := addr & 0x0007
		cart.state.registers.Fetcher[f].Low = data

	// fast fetch mode
	case 0x58:
		// ----------------------------------------
		//  Fast Fetch Mode
		// ----------------------------------------
		//  Fast Fetch Mode enables the fastest way to read DPC+ registers.  Normal
		//  reads use LDA Absolute addressing (LDA DF0DATA) which takes 4 cycles to
		//  process.  Fast Fetch Mode intercepts LDA Immediate addressing (LDA #<DF0DATA)
		//  which takes only 2 cycles!  Only immediate values < $28 are intercepted
		cart.state.registers.FastFetch = data == 0

	// function support - parameter
	case 0x59:

	// function support - call function
	case 0x5a:

	// reserved
	case 0x5b:
	case 0x5c:

	// waveforms
	case 0x5d:
		cart.state.registers.MusicFetcher[0].Waveform = uint32(data & 0x7f)
	case 0x5e:
		cart.state.registers.MusicFetcher[1].Waveform = uint32(data & 0x7f)
	case 0x5f:
		// ----------------------------------------
		//  Waveforms
		// ----------------------------------------
		//  Waveforms are 32 byte tables that define a waveform.  Waveforms must be 32
		//  byte aligned, and can only be stored in the 4K Display Data Bank. You MUST
		//  define an "OFF" waveform,  comprised of all zeros.  The sum of all waveforms
		//  being played should be <= 15, so typically you'll use a maximum of 5 for any
		//  given value.
		//
		//  Valid values are 0-127 and point to the 4K Display Data bank.  The formula
		//  (* & $1fff)/32 as shown below will calculate the value for you
		cart.state.registers.MusicFetcher[2].Waveform = uint32(data & 0x7f)

	// data fetcher, push stack
	case 0x60:
		fallthrough
	case 0x61:
		fallthrough
	case 0x62:
		fallthrough
	case 0x63:
		fallthrough
	case 0x64:
		fallthrough
	case 0x65:
		fallthrough
	case 0x66:
		fallthrough
	case 0x67:
		// ----------------------------------------
		//  Data Fetcher Push (stack)
		// ----------------------------------------
		//  The Data Fetchers can also be used to update the contents of the 4K
		//  Display Data bank.  Point the Data Fetcher to the data to change,
		//  then Push to it.  The Data Fetcher's pointer will be decremented BEFORE
		//  the data is written.
		f := addr & 0x0007
		cart.state.registers.Fetcher[f].dec()
		dataAddr := uint16(cart.state.registers.Fetcher[f].Hi)<<8 | uint16(cart.state.registers.Fetcher[f].Low)
		dataAddr &= 0x0fff
		cart.static.Data[dataAddr] = data

	// data fetcher, high pointer
	case 0x68:
		fallthrough
	case 0x69:
		fallthrough
	case 0x6a:
		fallthrough
	case 0x6b:
		fallthrough
	case 0x6c:
		fallthrough
	case 0x6d:
		fallthrough
	case 0x6e:
		fallthrough
	case 0x6f:
		f := addr & 0x0007
		cart.state.registers.Fetcher[f].Hi = data

	// random number initialisation
	case 0x70:
		cart.state.registers.RNG.Value = 0x2b435044
	case 0x71:
		cart.state.registers.RNG.Value &= 0xffffff00
		cart.state.registers.RNG.Value |= uint32(data)
	case 0x72:
		cart.state.registers.RNG.Value &= 0xffff00ff
		cart.state.registers.RNG.Value |= uint32(data) << 8
	case 0x73:
		cart.state.registers.RNG.Value &= 0xff00ffff
		cart.state.registers.RNG.Value |= uint32(data) << 16
	case 0x74:
		cart.state.registers.RNG.Value &= 0x00ffffff
		cart.state.registers.RNG.Value |= uint32(data) << 24

	// musical notes
	case 0x75:
		cart.state.registers.MusicFetcher[0].Freq = uint32(cart.static.Freq[data<<2])
		cart.state.registers.MusicFetcher[0].Freq += uint32(cart.static.Freq[(data<<2)+1]) << 8
		cart.state.registers.MusicFetcher[0].Freq += uint32(cart.static.Freq[(data<<2)+2]) << 16
		cart.state.registers.MusicFetcher[0].Freq += uint32(cart.static.Freq[(data<<2)+3]) << 24
	case 0x76:
		cart.state.registers.MusicFetcher[1].Freq = uint32(cart.static.Freq[data<<2])
		cart.state.registers.MusicFetcher[1].Freq += uint32(cart.static.Freq[(data<<2)+1]) << 8
		cart.state.registers.MusicFetcher[1].Freq += uint32(cart.static.Freq[(data<<2)+2]) << 16
		cart.state.registers.MusicFetcher[1].Freq += uint32(cart.static.Freq[(data<<2)+3]) << 24
	case 0x77:
		cart.state.registers.MusicFetcher[2].Freq = uint32(cart.static.Freq[data<<2])
		cart.state.registers.MusicFetcher[2].Freq += uint32(cart.static.Freq[(data<<2)+1]) << 8
		cart.state.registers.MusicFetcher[2].Freq += uint32(cart.static.Freq[(data<<2)+2]) << 16
		cart.state.registers.MusicFetcher[2].Freq += uint32(cart.static.Freq[(data<<2)+3]) << 24

	// data fetcher, queue
	case 0x78:
		fallthrough
	case 0x79:
		fallthrough
	case 0x7a:
		fallthrough
	case 0x7b:
		fallthrough
	case 0x7c:
		fallthrough
	case 0x7d:
		fallthrough
	case 0x7e:
		fallthrough
	case 0x7f:
		// ----------------------------------------
		//  Data Fetcher Write (queue)
		// ----------------------------------------
		//  The Data Fetchers can also be used to update the contents of the 4K
		//  Display Data bank.  Point the Data Fetcher to the data to change,
		//  then Write to it  The Data Fetcher's pointer will be incremented AFTER
		//  the data is written.
		f := addr & 0x0007
		dataAddr := uint16(cart.state.registers.Fetcher[f].Hi)<<8 | uint16(cart.state.registers.Fetcher[f].Low)
		dataAddr &= 0x0fff
		cart.static.Data[dataAddr] = data
		cart.state.registers.Fetcher[f].inc()
	}

	if poke {
		cart.banks[cart.bank][addr] = data
		return nil
	}

	return curated.Errorf("DPC+: %v", curated.Errorf(bus.AddressError, addr))
}

// bankswitch on hotspot access.
func (cart *dpcPlus) bankswitch(addr uint16, passive bool) bool {
	if addr >= 0x0ff6 && addr <= 0x0ffb {
		if passive {
			return true
		}
		if addr == 0x0ff6 {
			cart.bank = 0
		} else if addr == 0x0ff7 {
			cart.bank = 1
		} else if addr == 0x0ff8 {
			cart.bank = 2
		} else if addr == 0x0ff9 {
			cart.bank = 3
		} else if addr == 0x0ffa {
			cart.bank = 4
		} else if addr == 0x0ffb {
			cart.bank = 5
		}
		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface.
func (cart dpcPlus) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the mapper.CartMapper interface.
func (cart dpcPlus) GetBank(addr uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: cart.bank, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *dpcPlus) Patch(offset int, data uint8) error {
	if offset >= cart.fileSize {
		return curated.Errorf("DPC+: %v", fmt.Errorf("patch offset too high (%v)", offset))
	}

	if offset >= cart.freqOffset {
		cart.static.Freq[offset-cart.freqOffset] = data
	} else if offset >= cart.dataOffset {
		cart.static.Data[offset-cart.dataOffset] = data
	} else if offset >= cart.banksOffset {
		bank := offset / cart.bankSize
		offset %= cart.bankSize
		cart.banks[bank][offset] = data
	} else {
		cart.static.Arm[offset-cart.banksOffset] = data
	}

	return nil
}

// Listen implements the mapper.CartMapper interface.
func (cart *dpcPlus) Listen(addr uint16, data uint8) {
}

// Step implements the mapper.CartMapper interface.
func (cart *dpcPlus) Step() {
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
}

// IterateBank implements the mapper.CartMapper interface.
func (cart dpcPlus) CopyBanks() []mapper.BankContent {
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
func (cart dpcPlus) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff6: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff7: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1ff8: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x1ffa: {Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x1ffb: {Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
		0x1000: {Symbol: "RNG/next", Action: mapper.HotspotRegister},
		0x1001: {Symbol: "RNG/0", Action: mapper.HotspotRegister},
		0x1002: {Symbol: "RNG/1", Action: mapper.HotspotRegister},
		0x1003: {Symbol: "RNG/2", Action: mapper.HotspotRegister},
		0x1004: {Symbol: "RNG/3", Action: mapper.HotspotRegister},
		0x1005: {Symbol: "MUSIC", Action: mapper.HotspotRegister},
		0x1006: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x1007: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x1008: {Symbol: "DF0", Action: mapper.HotspotRegister},
		0x1009: {Symbol: "DF1", Action: mapper.HotspotRegister},
		0x100a: {Symbol: "DF2", Action: mapper.HotspotRegister},
		0x100b: {Symbol: "DF3", Action: mapper.HotspotRegister},
		0x100c: {Symbol: "DF4", Action: mapper.HotspotRegister},
		0x100d: {Symbol: "DF5", Action: mapper.HotspotRegister},
		0x100e: {Symbol: "DF6", Action: mapper.HotspotRegister},
		0x100f: {Symbol: "DF7", Action: mapper.HotspotRegister},
		0x1010: {Symbol: "DF0/win", Action: mapper.HotspotRegister},
		0x1011: {Symbol: "DF1/win", Action: mapper.HotspotRegister},
		0x1012: {Symbol: "DF2/win", Action: mapper.HotspotRegister},
		0x1013: {Symbol: "DF3/win", Action: mapper.HotspotRegister},
		0x1014: {Symbol: "DF4/win", Action: mapper.HotspotRegister},
		0x1015: {Symbol: "DF5/win", Action: mapper.HotspotRegister},
		0x1016: {Symbol: "DF6/win", Action: mapper.HotspotRegister},
		0x1017: {Symbol: "DF7/win", Action: mapper.HotspotRegister},
		0x1018: {Symbol: "DF0/frac", Action: mapper.HotspotRegister},
		0x1019: {Symbol: "DF1/frac", Action: mapper.HotspotRegister},
		0x101a: {Symbol: "DF2/frac", Action: mapper.HotspotRegister},
		0x101b: {Symbol: "DF3/frac", Action: mapper.HotspotRegister},
		0x101c: {Symbol: "DF4/frac", Action: mapper.HotspotRegister},
		0x101d: {Symbol: "DF5/frac", Action: mapper.HotspotRegister},
		0x101e: {Symbol: "DF6/frac", Action: mapper.HotspotRegister},
		0x101f: {Symbol: "DF7/frac", Action: mapper.HotspotRegister},
		0x1020: {Symbol: "ISWIN0", Action: mapper.HotspotRegister},
		0x1021: {Symbol: "ISWIN1", Action: mapper.HotspotRegister},
		0x1022: {Symbol: "ISWIN2", Action: mapper.HotspotRegister},
		0x1023: {Symbol: "ISWIN3", Action: mapper.HotspotRegister},
		0x1024: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x1025: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x1026: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x1027: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart dpcPlus) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff6: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff7: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1ff8: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x1ffa: {Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x1ffb: {Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
		0x1028: {Symbol: "FDF0/low", Action: mapper.HotspotRegister},
		0x1029: {Symbol: "FDF1/low", Action: mapper.HotspotRegister},
		0x102a: {Symbol: "FDF2/low", Action: mapper.HotspotRegister},
		0x102b: {Symbol: "FDF3/low", Action: mapper.HotspotRegister},
		0x102c: {Symbol: "FDF4/low", Action: mapper.HotspotRegister},
		0x102d: {Symbol: "FDF5/low", Action: mapper.HotspotRegister},
		0x102e: {Symbol: "FDF6/low", Action: mapper.HotspotRegister},
		0x102f: {Symbol: "FDF7/low", Action: mapper.HotspotRegister},
		0x1030: {Symbol: "FDF0/hi", Action: mapper.HotspotRegister},
		0x1031: {Symbol: "FDF1/hi", Action: mapper.HotspotRegister},
		0x1032: {Symbol: "FDF2/hi", Action: mapper.HotspotRegister},
		0x1033: {Symbol: "FDF3/hi", Action: mapper.HotspotRegister},
		0x1034: {Symbol: "FDF4/hi", Action: mapper.HotspotRegister},
		0x1035: {Symbol: "FDF5/hi", Action: mapper.HotspotRegister},
		0x1036: {Symbol: "FDF6/hi", Action: mapper.HotspotRegister},
		0x1037: {Symbol: "FDF7/hi", Action: mapper.HotspotRegister},
		0x1038: {Symbol: "FDF0/inc", Action: mapper.HotspotRegister},
		0x1039: {Symbol: "FDF1/inc", Action: mapper.HotspotRegister},
		0x103a: {Symbol: "FDF2/inc", Action: mapper.HotspotRegister},
		0x103b: {Symbol: "FDF3/inc", Action: mapper.HotspotRegister},
		0x103c: {Symbol: "FDF4/inc", Action: mapper.HotspotRegister},
		0x103d: {Symbol: "FDF5/inc", Action: mapper.HotspotRegister},
		0x103e: {Symbol: "FDF6/inc", Action: mapper.HotspotRegister},
		0x103f: {Symbol: "FDF7/inc", Action: mapper.HotspotRegister},
		0x1040: {Symbol: "DF0/top", Action: mapper.HotspotRegister},
		0x1041: {Symbol: "DF1/top", Action: mapper.HotspotRegister},
		0x1042: {Symbol: "DF2/top", Action: mapper.HotspotRegister},
		0x1043: {Symbol: "DF3/top", Action: mapper.HotspotRegister},
		0x1044: {Symbol: "DF4/top", Action: mapper.HotspotRegister},
		0x1045: {Symbol: "DF5/top", Action: mapper.HotspotRegister},
		0x1046: {Symbol: "DF6/top", Action: mapper.HotspotRegister},
		0x1047: {Symbol: "DF7/top", Action: mapper.HotspotRegister},
		0x1048: {Symbol: "DF0/bot", Action: mapper.HotspotRegister},
		0x1049: {Symbol: "DF1/bot", Action: mapper.HotspotRegister},
		0x104a: {Symbol: "DF2/bot", Action: mapper.HotspotRegister},
		0x104b: {Symbol: "DF3/bot", Action: mapper.HotspotRegister},
		0x104c: {Symbol: "DF4/bot", Action: mapper.HotspotRegister},
		0x104d: {Symbol: "DF5/bot", Action: mapper.HotspotRegister},
		0x104e: {Symbol: "DF6/bot", Action: mapper.HotspotRegister},
		0x104f: {Symbol: "DF7/bot", Action: mapper.HotspotRegister},
		0x1050: {Symbol: "DF0/low", Action: mapper.HotspotRegister},
		0x1051: {Symbol: "DF1/low", Action: mapper.HotspotRegister},
		0x1052: {Symbol: "DF2/low", Action: mapper.HotspotRegister},
		0x1053: {Symbol: "DF3/low", Action: mapper.HotspotRegister},
		0x1054: {Symbol: "DF4/low", Action: mapper.HotspotRegister},
		0x1055: {Symbol: "DF5/low", Action: mapper.HotspotRegister},
		0x1056: {Symbol: "DF6/low", Action: mapper.HotspotRegister},
		0x1057: {Symbol: "DF7/low", Action: mapper.HotspotRegister},
		0x1058: {Symbol: "FASTFETCH", Action: mapper.HotspotRegister},
		0x1059: {Symbol: "PARAM", Action: mapper.HotspotRegister},
		0x105a: {Symbol: "FUNC", Action: mapper.HotspotFunction},
		0x105b: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x105c: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x105d: {Symbol: "MF0", Action: mapper.HotspotRegister},
		0x105e: {Symbol: "MF1", Action: mapper.HotspotRegister},
		0x105f: {Symbol: "MF2", Action: mapper.HotspotRegister},
		0x1060: {Symbol: "DF0/push", Action: mapper.HotspotRegister},
		0x1061: {Symbol: "DF1/push", Action: mapper.HotspotRegister},
		0x1062: {Symbol: "DF2/push", Action: mapper.HotspotRegister},
		0x1063: {Symbol: "DF3/push", Action: mapper.HotspotRegister},
		0x1064: {Symbol: "DF4/push", Action: mapper.HotspotRegister},
		0x1065: {Symbol: "DF5/push", Action: mapper.HotspotRegister},
		0x1066: {Symbol: "DF6/push", Action: mapper.HotspotRegister},
		0x1067: {Symbol: "DF7/push", Action: mapper.HotspotRegister},
		0x1068: {Symbol: "DF0/hi", Action: mapper.HotspotRegister},
		0x1069: {Symbol: "DF1/hi", Action: mapper.HotspotRegister},
		0x106a: {Symbol: "DF2/hi", Action: mapper.HotspotRegister},
		0x106b: {Symbol: "DF3/hi", Action: mapper.HotspotRegister},
		0x106c: {Symbol: "DF4/hi", Action: mapper.HotspotRegister},
		0x106d: {Symbol: "DF5/hi", Action: mapper.HotspotRegister},
		0x106e: {Symbol: "DF6/hi", Action: mapper.HotspotRegister},
		0x106f: {Symbol: "DF7/hi", Action: mapper.HotspotRegister},
		0x1070: {Symbol: "RNGINIT", Action: mapper.HotspotFunction},
		0x1071: {Symbol: "RNG0", Action: mapper.HotspotRegister},
		0x1072: {Symbol: "RNG1", Action: mapper.HotspotRegister},
		0x1073: {Symbol: "RNG2", Action: mapper.HotspotRegister},
		0x1074: {Symbol: "RNG3", Action: mapper.HotspotRegister},
		0x1075: {Symbol: "MUSIC0", Action: mapper.HotspotRegister},
		0x1076: {Symbol: "MUSIC1", Action: mapper.HotspotRegister},
		0x1077: {Symbol: "MUSIC2", Action: mapper.HotspotRegister},
		0x1078: {Symbol: "DF0/queue", Action: mapper.HotspotRegister},
		0x1079: {Symbol: "DF1/queue", Action: mapper.HotspotRegister},
		0x107a: {Symbol: "DF2/queue", Action: mapper.HotspotRegister},
		0x107b: {Symbol: "DF3/queue", Action: mapper.HotspotRegister},
		0x107c: {Symbol: "DF4/queue", Action: mapper.HotspotRegister},
		0x107d: {Symbol: "DF5/queue", Action: mapper.HotspotRegister},
		0x107e: {Symbol: "DF6/queue", Action: mapper.HotspotRegister},
		0x107f: {Symbol: "DF7/queue", Action: mapper.HotspotRegister},
	}
}
