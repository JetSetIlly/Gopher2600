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

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
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

	registers DPCplusRegisters
	static    DPCplusStatic

	// was the last instruction read the opcode for "lda <immediate>"
	lda bool

	// music fetchers are clocked at a fixed (slower) rate than the reference
	// to the VCS's clock. see Step() function.
	beats int

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

// NewDPCplus is the preferred method of initialisation for the harmony type
func NewDPCplus(data []byte) (*dpcPlus, error) {
	const armSize = 3072
	const dataSize = 4096
	const freqSize = 1024

	cart := &dpcPlus{
		mappingID:   "DPC+",
		description: "harmony",
		bankSize:    4096,
	}

	// amount of data used for cartridges
	bankLen := len(data) - dataSize - armSize - freqSize

	// size check
	if bankLen <= 0 || bankLen%cart.bankSize != 0 {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in cartridge data", cart.mappingID))
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

	// initialise cartridge before returning success
	cart.Initialise()

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

func (cart dpcPlus) ID() string {
	return cart.mappingID
}

func (cart *dpcPlus) Initialise() {
	cart.bank = len(cart.banks) - 1
}

func (cart *dpcPlus) Read(addr uint16, passive bool) (uint8, error) {
	if cart.hotspot(addr, passive) {
		// always return zero on hotspot - unlike the Atari multi-bank carts for example
		return 0, nil
	}

	var data uint8

	// if address is above register space then we only need to check for bank
	// switching before returning data at the quoted address
	if addr > 0x007f {
		data = cart.banks[cart.bank][addr]

		// if FastFetch mode is on and the preceeding data value was 0xa9 (the
		// opcode for LDA <immediate>) then the data we've just read this cycle
		// should be interpreted as an address to read from. we can do this by
		// recursing into the Read() function (there is no worry about deep
		// recursions because we reset the lda flag before recursing and the
		// lda flag being set is a prerequisite for the recursion to take
		// place)
		if cart.registers.FastFetch && cart.lda && data < 0x28 {
			cart.lda = false
			return cart.Read(uint16(data), passive)
		} else {
			cart.lda = cart.registers.FastFetch && data == 0xa9
			return data, nil
		}
	}

	if addr > 0x0027 {
		return 0, errors.New(errors.MemoryBusError, addr)
	}

	switch addr {
	// random number generator
	case 0x00:
		cart.registers.RNG.next()
		data = uint8(cart.registers.RNG.Value)
	case 0x01:
		cart.registers.RNG.prev()
		data = uint8(cart.registers.RNG.Value)
	case 0x02:
		data = uint8(cart.registers.RNG.Value >> 8)
	case 0x03:
		data = uint8(cart.registers.RNG.Value >> 16)
	case 0x04:
		data = uint8(cart.registers.RNG.Value >> 24)

	// music fetcher
	case 0x05:
		data = cart.static.Data[(cart.registers.MusicFetcher[0].Waveform<<5)+(cart.registers.MusicFetcher[0].Count>>27)]
		data += cart.static.Data[(cart.registers.MusicFetcher[1].Waveform<<5)+(cart.registers.MusicFetcher[1].Count>>27)]
		data += cart.static.Data[(cart.registers.MusicFetcher[2].Waveform<<5)+(cart.registers.MusicFetcher[2].Count>>27)]

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
		dataAddr := uint16(cart.registers.Fetcher[f].Hi)<<8 | uint16(cart.registers.Fetcher[f].Low)
		dataAddr = dataAddr & 0x0fff
		data = cart.static.Data[dataAddr]
		cart.registers.Fetcher[f].inc()

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
		dataAddr := uint16(cart.registers.Fetcher[f].Hi)<<8 | uint16(cart.registers.Fetcher[f].Low)
		dataAddr = dataAddr & 0x0fff
		if cart.registers.Fetcher[f].isWindow() {
			data = cart.static.Data[dataAddr]
		}
		cart.registers.Fetcher[f].inc()

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
		dataAddr := uint16(cart.registers.FracFetcher[f].Hi)<<8 | uint16(cart.registers.FracFetcher[f].Low)
		dataAddr = dataAddr & 0x0fff
		data = cart.static.Data[dataAddr]
		cart.registers.FracFetcher[f].inc()

	// data fetcher window flag
	case 0x20:
		fallthrough
	case 0x21:
		fallthrough
	case 0x22:
		fallthrough
	case 0x23:
		f := addr & 0x0007
		if cart.registers.Fetcher[f].isWindow() {
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

func (cart *dpcPlus) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if cart.hotspot(addr, passive) {
		return nil
	}

	if addr < 0x0028 || addr > 0x007f {
		return errors.New(errors.MemoryBusError, addr)
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
		cart.registers.FracFetcher[f].Low = data
		cart.registers.FracFetcher[f].Count = 0

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
		cart.registers.FracFetcher[f].Hi = data
		cart.registers.FracFetcher[f].Count = 0

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
		cart.registers.FracFetcher[f].Increment = data
		cart.registers.FracFetcher[f].Count = 0

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
		cart.registers.Fetcher[f].Top = data

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
		cart.registers.Fetcher[f].Bottom = data

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
		cart.registers.Fetcher[f].Low = data

	// fast fetch mode
	case 0x58:
		// ----------------------------------------
		//  Fast Fetch Mode
		// ----------------------------------------
		//  Fast Fetch Mode enables the fastest way to read DPC+ registers.  Normal
		//  reads use LDA Absolute addressing (LDA DF0DATA) which takes 4 cycles to
		//  process.  Fast Fetch Mode intercepts LDA Immediate addressing (LDA #<DF0DATA)
		//  which takes only 2 cycles!  Only immediate values < $28 are intercepted
		cart.registers.FastFetch = data == 0

	// function support - parameter
	case 0x59:

	// function support - call function
	case 0x5a:

	// reserved
	case 0x5b:
	case 0x5c:

	// waveforms
	case 0x5d:
		cart.registers.MusicFetcher[0].Waveform = uint32(data & 0x7f)
	case 0x5e:
		cart.registers.MusicFetcher[1].Waveform = uint32(data & 0x7f)
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
		cart.registers.MusicFetcher[2].Waveform = uint32(data & 0x7f)

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
		cart.registers.Fetcher[f].dec()
		dataAddr := uint16(cart.registers.Fetcher[f].Hi)<<8 | uint16(cart.registers.Fetcher[f].Low)
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
		cart.registers.Fetcher[f].Hi = data

	// random number initialisation
	case 0x70:
		cart.registers.RNG.Value = 0x2b435044
	case 0x71:
		cart.registers.RNG.Value &= 0xffffff00
		cart.registers.RNG.Value |= uint32(data)
	case 0x72:
		cart.registers.RNG.Value &= 0xffff00ff
		cart.registers.RNG.Value |= uint32(data) << 8
	case 0x73:
		cart.registers.RNG.Value &= 0xff00ffff
		cart.registers.RNG.Value |= uint32(data) << 16
	case 0x74:
		cart.registers.RNG.Value &= 0x00ffffff
		cart.registers.RNG.Value |= uint32(data) << 24

	// musical notes
	case 0x75:
		cart.registers.MusicFetcher[0].Freq = uint32(cart.static.Freq[data<<2])
		cart.registers.MusicFetcher[0].Freq += uint32(cart.static.Freq[(data<<2)+1]) << 8
		cart.registers.MusicFetcher[0].Freq += uint32(cart.static.Freq[(data<<2)+2]) << 16
		cart.registers.MusicFetcher[0].Freq += uint32(cart.static.Freq[(data<<2)+3]) << 24
	case 0x76:
		cart.registers.MusicFetcher[1].Freq = uint32(cart.static.Freq[data<<2])
		cart.registers.MusicFetcher[1].Freq += uint32(cart.static.Freq[(data<<2)+1]) << 8
		cart.registers.MusicFetcher[1].Freq += uint32(cart.static.Freq[(data<<2)+2]) << 16
		cart.registers.MusicFetcher[1].Freq += uint32(cart.static.Freq[(data<<2)+3]) << 24
	case 0x77:
		cart.registers.MusicFetcher[2].Freq = uint32(cart.static.Freq[data<<2])
		cart.registers.MusicFetcher[2].Freq += uint32(cart.static.Freq[(data<<2)+1]) << 8
		cart.registers.MusicFetcher[2].Freq += uint32(cart.static.Freq[(data<<2)+2]) << 16
		cart.registers.MusicFetcher[2].Freq += uint32(cart.static.Freq[(data<<2)+3]) << 24

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
		dataAddr := uint16(cart.registers.Fetcher[f].Hi)<<8 | uint16(cart.registers.Fetcher[f].Low)
		dataAddr &= 0x0fff
		cart.static.Data[dataAddr] = data
		cart.registers.Fetcher[f].inc()
	}

	if poke {
		cart.banks[cart.bank][addr] = data
		return nil
	}

	return errors.New(errors.MemoryBusError, addr)
}

// bankswitch on hotspot access
func (cart *dpcPlus) hotspot(addr uint16, passive bool) bool {
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

func (cart dpcPlus) NumBanks() int {
	return len(cart.banks)
}

func (cart dpcPlus) GetBank(addr uint16) banks.Details {
	return banks.Details{Number: cart.bank, IsRAM: false}
}

func (cart *dpcPlus) Patch(offset int, data uint8) error {
	if offset >= cart.fileSize {
		return errors.New(errors.CartridgePatchOOB, offset)
	}

	if offset >= cart.freqOffset {
		cart.static.Freq[offset-cart.freqOffset] = data
	} else if offset >= cart.dataOffset {
		cart.static.Data[offset-cart.dataOffset] = data
	} else if offset >= cart.banksOffset {
		bank := int(offset) / cart.bankSize
		offset = offset % cart.bankSize
		cart.banks[bank][offset] = data
	} else {
		cart.static.Arm[offset-cart.banksOffset] = data
	}

	return nil
}

func (cart *dpcPlus) Listen(addr uint16, data uint8) {
}

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

	cart.beats++
	if cart.beats%59 == 0 {
		cart.beats = 0
		cart.registers.MusicFetcher[0].Count += cart.registers.MusicFetcher[0].Freq
		cart.registers.MusicFetcher[1].Count += cart.registers.MusicFetcher[1].Freq
		cart.registers.MusicFetcher[2].Count += cart.registers.MusicFetcher[2].Freq
	}
}

// IterateBank implemnts the disassemble interface
func (cart dpcPlus) IterateBanks(prev *banks.Content) *banks.Content {
	b := prev.Number + 1
	if b < len(cart.banks) {
		return &banks.Content{Number: b,
			Data: cart.banks[b],
			Origins: []uint16{
				memorymap.OriginCart,
			},
		}
	}
	return nil
}
