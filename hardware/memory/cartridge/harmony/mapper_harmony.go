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
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// harmony implements the cartMapper interface.
//
// https://atariage.com/forums/topic/163495-harmony-dpc-programming
type harmony struct {
	mappingID   string
	description string

	arm   []byte
	banks [][]byte
	data  []byte
	freq  []byte

	// the currently selected bank
	bank int

	fetcher      [8]dataFetcher
	fracFetcher  [8]fractionalDataFetcher
	musicFetcher [3]musicDataFetcher
	window       [4]bool

	// random number generator
	rng randomNumberFetcher

	// fast fetch read mode
	fastFatch bool

	// was the last instruction read the opcode for "lda <immediate>"
	lda bool

	// music fetchers are clocked at a fixed (slower) rate than the reference
	// to the VCS's clock. see Step() function.
	beats int
}

type randomNumberFetcher struct {
	value uint32
}

func (rng *randomNumberFetcher) next() {
	if rng.value&(1<<10) != 0 {
		rng.value = 0x10adab1e ^ ((rng.value >> 11) | (rng.value << 21))
	} else {
		rng.value = 0x00 ^ ((rng.value >> 11) | (rng.value << 21))
	}
}

func (rng *randomNumberFetcher) prev() {
	if rng.value&(1<<31) != 0 {
		rng.value = ((0x10adab1e & rng.value) << 11) | ((0x10adab1e ^ rng.value) >> 21)
	} else {
		rng.value = (rng.value << 11) | (rng.value >> 21)
	}
}

type musicDataFetcher struct {
	waveform uint32
	freq     uint32
	count    uint32
}

type dataFetcher struct {
	low byte
	hi  byte

	top    byte
	bottom byte
}

func (df *dataFetcher) isWindow() bool {
	// unlike the original DPC format checing to see if a data fetcher is in
	// its window has to be done on demand. it has to be like this because the
	// demo ROMs that show off the DPC+ format require it. to put it simply, if
	// we implemented the window flag is it is described in the DPC patent then
	// the DPC+ demo ROMs would miss the window by setting the low attribute
	// toa high (ie. beyond the top value) for the window to caught in the
	// flag->true condition.

	if df.top > df.bottom {
		return df.low > df.top || df.low < df.bottom
	}
	return df.low > df.top && df.low < df.bottom
}

func (df *dataFetcher) inc() {
	df.low++
	if df.low == 0x00 {
		df.hi++
	}
}

func (df *dataFetcher) dec() {
	df.low--
	if df.low == 0x00 {
		df.hi--
	}
}

type fractionalDataFetcher struct {
	low byte
	hi  byte

	increment byte
	count     byte
}

func (df *fractionalDataFetcher) inc() {
	df.count += df.increment
	if df.count < df.increment {
		df.low++
		if df.low == 0x00 {
			df.hi++
		}
	}
}

// NewHarmony is the preferred method of initialisation for the harmony type
func NewHarmony(data []byte) (*harmony, error) {
	const armSize = 3072
	const bankSize = 4096
	const dataSize = 4096
	const freqSize = 1024

	cart := &harmony{}
	cart.mappingID = "DPC+"
	cart.description = "DPC+ (Harmony)"
	cart.banks = make([][]uint8, cart.NumBanks())

	// amount of data used for cartridges
	bankLen := len(data) - dataSize - armSize - freqSize

	// size check
	if bankLen%bankSize != 0 {
		return nil, errors.New(errors.HarmonyError, fmt.Sprintf("%d bytes not supported", len(data)))
	}

	// partition
	cart.arm = data[:armSize]

	// allocate enough banks
	cart.banks = make([][]uint8, bankLen/bankSize)

	// partition data into banks
	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, bankSize)
		offset := k * bankSize
		offset += armSize
		cart.banks[k] = data[offset : offset+bankSize]
	}

	// gfx and frequency table at end of file
	s := armSize + (bankSize * cart.NumBanks())
	cart.data = data[s : s+dataSize]
	cart.freq = data[s+dataSize:]

	// initialise cartridge before returning success
	cart.Initialise()

	return cart, nil
}

func (cart harmony) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.description, cart.mappingID, cart.bank)
}

func (cart harmony) ID() string {
	return cart.mappingID
}

func (cart *harmony) Initialise() {
	cart.bank = len(cart.banks) - 1
}

func (cart *harmony) Read(addr uint16) (uint8, error) {
	var data uint8

	// if address is above register space then we only need to check for bank
	// switching before returning data at the quoted address
	if addr > 0x007f {
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
		} else {
			data = cart.banks[cart.bank][addr]
		}

		// if fastFetch mode is on and the preceeding data value was 0xa9 (the
		// opcode for LDA <immediate>) then the data we've just read this cycle
		// should be interpreted as an address to read from. we can do this by
		// recursing into the Read() function (there is no worry about deep
		// recursions because we reset the lda flag before recursing and the
		// lda flag being set is a prerequisite for the recursion to take
		// place)
		if cart.fastFatch && cart.lda && data < 0x28 {
			cart.lda = false
			return cart.Read(uint16(data))
		} else {
			cart.lda = cart.fastFatch && data == 0xa9
			return data, nil
		}
	}

	if addr > 0x0027 {
		return 0, errors.New(errors.BusError, addr)
	}

	switch addr {
	// random number generator
	case 0x00:
		cart.rng.next()
		data = uint8(cart.rng.value)
	case 0x01:
		cart.rng.prev()
		data = uint8(cart.rng.value)
	case 0x02:
		data = uint8(cart.rng.value >> 8)
	case 0x03:
		data = uint8(cart.rng.value >> 16)
	case 0x04:
		data = uint8(cart.rng.value >> 24)

	// music fetcher
	case 0x05:
		data = cart.data[(cart.musicFetcher[0].waveform<<5)+(cart.musicFetcher[0].count>>27)]
		data += cart.data[(cart.musicFetcher[1].waveform<<5)+(cart.musicFetcher[1].count>>27)]
		data += cart.data[(cart.musicFetcher[2].waveform<<5)+(cart.musicFetcher[2].count>>27)]

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
		dataAddr := uint16(cart.fetcher[f].hi)<<8 | uint16(cart.fetcher[f].low)
		dataAddr = dataAddr & 0x0fff
		data = cart.data[dataAddr]
		cart.fetcher[f].inc()

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
		dataAddr := uint16(cart.fetcher[f].hi)<<8 | uint16(cart.fetcher[f].low)
		dataAddr = dataAddr & 0x0fff
		if cart.fetcher[f].isWindow() {
			data = cart.data[dataAddr]
		}
		cart.fetcher[f].inc()

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
		dataAddr := uint16(cart.fracFetcher[f].hi)<<8 | uint16(cart.fracFetcher[f].low)
		dataAddr = dataAddr & 0x0fff
		data = cart.data[dataAddr]
		cart.fracFetcher[f].inc()

	// data fetcher window flag
	case 0x20:
		fallthrough
	case 0x21:
		fallthrough
	case 0x22:
		fallthrough
	case 0x23:
		f := addr & 0x0007
		if cart.fetcher[f].isWindow() {
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

func (cart *harmony) Write(addr uint16, data uint8) error {
	// if address is above register space then we only need to check for bank
	// switching before returning data at the quoted address
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

	if addr < 0x0028 || addr > 0x007f {
		return errors.New(errors.BusError, addr)
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
		cart.fracFetcher[f].low = data

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
		cart.fracFetcher[f].hi = data

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
		cart.fracFetcher[f].increment = data
		cart.fracFetcher[f].count = data

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
		cart.fetcher[f].top = data

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
		cart.fetcher[f].bottom = data

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
		cart.fetcher[f].low = data

	// fast fetch mode
	case 0x58:
		// ----------------------------------------
		//  Fast Fetch Mode
		// ----------------------------------------
		//  Fast Fetch Mode enables the fastest way to read DPC+ registers.  Normal
		//  reads use LDA Absolute addressing (LDA DF0DATA) which takes 4 cycles to
		//  process.  Fast Fetch Mode intercepts LDA Immediate addressing (LDA #<DF0DATA)
		//  which takes only 2 cycles!  Only immediate values < $28 are intercepted
		cart.fastFatch = data == 0

	// function support - parameter
	case 0x59:

	// function support - call function
	case 0x5a:

	// reserved
	case 0x5b:
	case 0x5c:

	// waveforms
	case 0x5d:
		cart.musicFetcher[0].waveform = uint32(data & 0x7f)
	case 0x5e:
		cart.musicFetcher[1].waveform = uint32(data & 0x7f)
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
		cart.musicFetcher[2].waveform = uint32(data & 0x7f)

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
		cart.fetcher[f].dec()
		dataAddr := uint16(cart.fetcher[f].hi)<<8 | uint16(cart.fetcher[f].low)
		dataAddr &= 0x0fff
		cart.data[dataAddr] = data

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
		cart.fetcher[f].hi = data

	// random number initialisation
	case 0x70:
		cart.rng.value = 0x2b435044
	case 0x71:
		cart.rng.value &= 0xffffff00
		cart.rng.value |= uint32(data)
	case 0x72:
		cart.rng.value &= 0xffff00ff
		cart.rng.value |= uint32(data) << 8
	case 0x73:
		cart.rng.value &= 0xff00ffff
		cart.rng.value |= uint32(data) << 16
	case 0x74:
		cart.rng.value &= 0x00ffffff
		cart.rng.value |= uint32(data) << 24

	// musical notes
	case 0x75:
		cart.musicFetcher[0].freq = uint32(cart.freq[data<<2])
		cart.musicFetcher[0].freq += uint32(cart.freq[(data<<2)+1]) << 8
		cart.musicFetcher[0].freq += uint32(cart.freq[(data<<2)+2]) << 16
		cart.musicFetcher[0].freq += uint32(cart.freq[(data<<2)+3]) << 24
	case 0x76:
		cart.musicFetcher[1].freq = uint32(cart.freq[data<<2])
		cart.musicFetcher[1].freq += uint32(cart.freq[(data<<2)+1]) << 8
		cart.musicFetcher[1].freq += uint32(cart.freq[(data<<2)+2]) << 16
		cart.musicFetcher[1].freq += uint32(cart.freq[(data<<2)+3]) << 24
	case 0x77:
		cart.musicFetcher[2].freq = uint32(cart.freq[data<<2])
		cart.musicFetcher[2].freq += uint32(cart.freq[(data<<2)+1]) << 8
		cart.musicFetcher[2].freq += uint32(cart.freq[(data<<2)+2]) << 16
		cart.musicFetcher[2].freq += uint32(cart.freq[(data<<2)+3]) << 24

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
		dataAddr := uint16(cart.fetcher[f].hi)<<8 | uint16(cart.fetcher[f].low)
		dataAddr &= 0x0fff
		cart.data[dataAddr] = data
		cart.fetcher[f].inc()
	}

	return nil
}

func (cart harmony) NumBanks() int {
	return len(cart.banks)
}

func (cart *harmony) SetBank(addr uint16, bank int) error {
	cart.bank = bank
	return nil
}

func (cart harmony) GetBank(addr uint16) int {
	return cart.bank
}

func (cart *harmony) SaveState() interface{} {
	return nil
}

func (cart *harmony) RestoreState(state interface{}) error {
	return nil
}

func (cart *harmony) Poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

func (cart *harmony) Patch(addr uint16, data uint8) error {
	return errors.New(errors.UnpatchableCartType, cart.description)
}

func (cart *harmony) Listen(addr uint16, data uint8) {
}

func (cart *harmony) Step() {
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
		cart.musicFetcher[0].count += cart.musicFetcher[0].freq
		cart.musicFetcher[1].count += cart.musicFetcher[1].freq
		cart.musicFetcher[2].count += cart.musicFetcher[2].freq
	}
}

func (cart harmony) GetRAM() []memorymap.SubArea {
	return nil
}

// StaticRead implements the StaticArea interface
func (cart harmony) StaticRead(addr uint16) (uint8, error) {
	if int(addr) >= len(cart.data) {
		return 0, errors.New(errors.CartridgeStaticOOB, addr)
	}

	return cart.data[addr], nil
}

// StaticWrite implements the StaticArea interface
func (cart *harmony) StaticWrite(addr uint16, data uint8) error {
	if int(addr) >= len(cart.data) {
		return errors.New(errors.CartridgeStaticOOB, addr)
	}
	cart.data[addr] = data
	return nil
}

// StaticSize implements the StaticArea interface
func (cart harmony) StaticSize() int {
	return len(cart.data)
}
