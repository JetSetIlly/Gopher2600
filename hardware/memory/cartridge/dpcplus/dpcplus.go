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

package dpcplus

import (
	"crypto/md5"
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/random"
)

// dpcPlus implements the mapper.CartMapper interface.
//
// https://atariage.com/forums/topic/163495-harmony-dpc-programming
//
// https://atariage.com/forums/blogs/entry/11811-dpcarm-part-6-dpc-cartridge-layout/
type dpcPlus struct {
	env       *environment.Environment
	mappingID string

	// additional CPU - used by some ROMs
	arm *arm.ARM

	// the hook that handles cartridge yields
	yieldHook coprocessor.CartYieldHook

	// there is only one version of DPC+ currently but this method of
	// specifying addresses mirrors how we do it in the CDF type
	version mmap

	// banks and the currently selected bank
	bankSize int
	banks    [][]byte

	// rewindable state
	state *State

	// armState is a copy of the ARM's state at the moment of the most recent
	// Snapshot. it's used only suring a Plumb() operation
	armState *arm.ARMState
}

// the sizes of these areas in a DPC+ cartridge are fixed. the custom arm code
// and the 6507 program fit around these sizes.
const (
	driverSize = 3072 // 3k
	dataSize   = 4096 // 4k
	freqSize   = 1024 // 1k
)

// NewDPCplus is the preferred method of initialisation for the dpcPlus type.
func NewDPCplus(env *environment.Environment, version string) (mapper.CartMapper, error) {
	data, err := io.ReadAll(env.Loader)
	if err != nil {
		return nil, fmt.Errorf("DPC+: %w", err)
	}

	cart := &dpcPlus{
		env:       env,
		mappingID: version,
		bankSize:  4096,
		state:     newDPCPlusState(),
		yieldHook: coprocessor.StubCartYieldHook{},
	}

	// set driver specific options for DPC+
	if version == "DPC+" {
		driverMD5 := fmt.Sprintf("%x", md5.Sum(data[:0xc00]))
		if !cart.state.setDriverSpecificOptions(driverMD5) {
			logger.Logf(cart.env, cart.mappingID, "unrecognised driver: %s", driverMD5)
		}
	}

	// report on driver specific options
	if cart.state.resetFracFetcherCounterWhenLowFieldIsSet {
		logger.Logf(cart.env, cart.mappingID, "fractional fetcher counter will be reset on setting of low byte")
	}

	// create addresses
	cart.version, err = newVersion(version)
	if err != nil {
		return nil, fmt.Errorf("DPC+: %s", err.Error())
	}

	// amount of data used for cartridges
	bankLen := len(data) - dataSize - driverSize - freqSize

	// size check
	if bankLen <= 0 || bankLen%cart.bankSize != 0 {
		return nil, fmt.Errorf("DPC+: wrong number of bytes in cartridge data")
	}

	// allocate enough banks
	cart.banks = make([][]uint8, bankLen/cart.bankSize)

	// partition data into banks
	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		offset += driverSize
		cart.banks[k] = data[offset : offset+cart.bankSize]
	}

	// initialise static memory
	cart.state.static, err = cart.newDPCplusStatic(cart.version, data)
	if err != nil {
		return nil, fmt.Errorf("DPC+: %s", err.Error())
	}

	// initialise ARM processor
	//
	// if bank0 has any ARM code then it will start at offset 0x08. first eight
	// bytes are the ARM header
	cart.arm = arm.NewARM(cart.env, cart.version.arch, cart.state.static, cart)

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *dpcPlus) MappedBanks() string {
	return fmt.Sprintf("Bank: %d", cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart *dpcPlus) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *dpcPlus) Snapshot() mapper.CartMapper {
	n := *cart

	// taking a snapshot of ARM state via the ARM itself can cause havoc if
	// this instance of the cart is not current (because the ARM pointer itself
	// may be stale or pointing to another emulation)
	if cart.armState == nil {
		n.armState = cart.arm.Snapshot()
	} else {
		n.armState = cart.armState.Snapshot()
	}

	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *dpcPlus) Plumb(env *environment.Environment) {
	cart.env = env
	if cart.armState == nil {
		panic("cannot plumb this ELF instance because the ARM state is nil")
	}
	cart.arm.Plumb(cart.env, cart.armState, cart.state.static, cart)
	cart.armState = nil
}

// Plumb implements the mapper.CartMapper interface.
func (cart *dpcPlus) PlumbFromDifferentEmulation(env *environment.Environment) {
	cart.env = env
	if cart.armState == nil {
		panic("cannot plumb this ELF instance because the ARM state is nil")
	}
	cart.arm = arm.NewARM(cart.env, cart.version.arch, cart.state.static, cart)
	cart.arm.Plumb(cart.env, cart.armState, cart.state.static, cart)
	cart.armState = nil
	cart.yieldHook = &coprocessor.StubCartYieldHook{}
}

// Reset implements the mapper.CartMapper interface.
func (cart *dpcPlus) Reset() error {
	var rnd *random.Random
	if cart.env.Prefs.RandomState.Get().(bool) {
		rnd = cart.env.Random
	}
	cart.state.initialise(cart.version, rnd)
	cart.SetBank("AUTO")
	return nil
}

// Access implements the mapper.CartMapper interface.
func (cart *dpcPlus) Access(addr uint16, peek bool) (uint8, uint8, error) {
	if b, ok := cart.state.callfn.Check(addr); ok {
		return b, mapper.CartDrivenPins, nil
	}

	if !peek {
		if cart.bankswitch(addr) {
			return 0, mapper.CartDrivenPins, nil
		}
	}

	var data uint8

	// if address is above register space then we only need to check for bank
	// switching before returning data at the quoted address
	if addr > 0x007f {
		data = cart.banks[cart.state.bank][addr]

		// if FastFetch mode is on and the preceding data value was 0xa9 (the
		// opcode for LDA <immediate>) then the data we've just read this cycle
		// should be interpreted as an address to read from. we can do this by
		// recursing into the Read() function (there is no worry about deep
		// recursions because we reset the lda flag before recursing and the
		// lda flag being set is a prerequisite for the recursion to take
		// place)
		if cart.state.registers.FastFetch && cart.state.lda && data < 0x28 {
			cart.state.lda = false
			return cart.Access(uint16(data), peek)
		}

		cart.state.lda = cart.state.registers.FastFetch && data == 0xa9
		return data, mapper.CartDrivenPins, nil
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
		data = cart.state.static.dataRAM.data[(cart.state.registers.MusicFetcher[0].Waveform<<5)+(cart.state.registers.MusicFetcher[0].Count>>27)]
		data += cart.state.static.dataRAM.data[(cart.state.registers.MusicFetcher[1].Waveform<<5)+(cart.state.registers.MusicFetcher[1].Count>>27)]
		data += cart.state.static.dataRAM.data[(cart.state.registers.MusicFetcher[2].Waveform<<5)+(cart.state.registers.MusicFetcher[2].Count>>27)]

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
		data = cart.state.static.dataRAM.data[dataAddr]
		if !peek {
			cart.state.registers.Fetcher[f].inc()
		}

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
			data = cart.state.static.dataRAM.data[dataAddr]
		}
		if !peek {
			cart.state.registers.Fetcher[f].inc()
		}

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
		data = cart.state.static.dataRAM.data[dataAddr]
		if !peek {
			cart.state.registers.FracFetcher[f].inc()
		}

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

	return data, mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *dpcPlus) AccessVolatile(addr uint16, data uint8, poke bool) error {
	// bank switches can not take place if coprocessor is active
	if cart.state.callfn.IsActive() {
		return nil
	}

	if !poke {
		if cart.bankswitch(addr) {
			return nil
		}
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

		// frac fetcher count is *sometimes* reset when the low byte is set.
		// depending on the specific version of the DPC+ driver being used
		if cart.state.resetFracFetcherCounterWhenLowFieldIsSet {
			cart.state.registers.FracFetcher[f].Count = 0
		}

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
		cart.state.registers.FracFetcher[f].Hi = data & 0x0f
		// frac fetcher count not reset when high byte is set

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
		// frac fetcher count not reset when increment is set
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
		cart.state.parameters = append(cart.state.parameters, data)

	// function support - call function
	case 0x5a:
		switch data {
		case 0:
			cart.state.parameters = cart.state.parameters[:0]
		case 1:
			// copy rom to fetcher
			if len(cart.state.parameters) != 4 {
				logger.Logf(cart.env, cart.mappingID, "wrong number of parameters for function call [%02x]", data)
				break // switch data
			}

			addr := (uint16(cart.state.parameters[1]) << 8) | uint16(cart.state.parameters[0])
			for i := uint8(0); i < cart.state.parameters[3]; i++ {
				f := cart.state.registers.Fetcher[cart.state.parameters[2]&0x07]
				o := uint16(f.Low) | (uint16(f.Hi) << 8) + uint16(i)

				// copying from data ROM to data RAM
				idx := uint16(i) + addr - uint16(len(cart.state.static.customROM.data))
				cart.state.static.dataRAM.data[o] = cart.state.static.dataROM.data[idx]
			}
			cart.state.parameters = cart.state.parameters[:0]
		case 2:
			// copy value to fetcher
			if len(cart.state.parameters) != 4 {
				logger.Logf(cart.env, cart.mappingID, "wrong number of parameters for function call [%02x]", data)
				break // switch data
			}

			for i := uint8(0); i < cart.state.parameters[3]; i++ {
				f := cart.state.registers.Fetcher[cart.state.parameters[2]&0x07]
				o := uint16(f.Low+i) | (uint16(f.Hi+i) << 8)
				cart.state.static.dataRAM.data[o] = cart.state.parameters[0]
			}

			cart.state.parameters = cart.state.parameters[:0]

		case 254:
			fallthrough
		case 255:
			runArm := func() {
				cart.arm.StartProfiling()
				defer cart.arm.ProcessProfiling()
				cart.state.yield = cart.runArm()
			}

			// keep calling runArm() for as long as program has not ended
			runArm()
			for cart.state.yield.Type != coprocessor.YieldProgramEnded {
				// the ARM should never return YieldSyncWithVCS when executing code
				// from the DPC+ type. if it does then it is an error and we should yield
				// with YieldExecutionError
				if cart.state.yield.Type == coprocessor.YieldSyncWithVCS {
					cart.state.yield.Type = coprocessor.YieldExecutionError
					cart.state.yield.Error = fmt.Errorf("DPC+ does not support SyncWithVCS yield type")
				}

				if cart.yieldHook.CartYield(cart.state.yield) == coprocessor.YieldHookEnd {
					break
				}
				runArm()
			}
		}

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
		cart.state.static.dataRAM.data[dataAddr] = data

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
		cart.state.registers.MusicFetcher[0].Freq = uint32(cart.state.static.freqRAM.data[data<<2])
		cart.state.registers.MusicFetcher[0].Freq += uint32(cart.state.static.freqRAM.data[(data<<2)+1]) << 8
		cart.state.registers.MusicFetcher[0].Freq += uint32(cart.state.static.freqRAM.data[(data<<2)+2]) << 16
		cart.state.registers.MusicFetcher[0].Freq += uint32(cart.state.static.freqRAM.data[(data<<2)+3]) << 24
	case 0x76:
		cart.state.registers.MusicFetcher[1].Freq = uint32(cart.state.static.freqRAM.data[data<<2])
		cart.state.registers.MusicFetcher[1].Freq += uint32(cart.state.static.freqRAM.data[(data<<2)+1]) << 8
		cart.state.registers.MusicFetcher[1].Freq += uint32(cart.state.static.freqRAM.data[(data<<2)+2]) << 16
		cart.state.registers.MusicFetcher[1].Freq += uint32(cart.state.static.freqRAM.data[(data<<2)+3]) << 24
	case 0x77:
		cart.state.registers.MusicFetcher[2].Freq = uint32(cart.state.static.freqRAM.data[data<<2])
		cart.state.registers.MusicFetcher[2].Freq += uint32(cart.state.static.freqRAM.data[(data<<2)+1]) << 8
		cart.state.registers.MusicFetcher[2].Freq += uint32(cart.state.static.freqRAM.data[(data<<2)+2]) << 16
		cart.state.registers.MusicFetcher[2].Freq += uint32(cart.state.static.freqRAM.data[(data<<2)+3]) << 24

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
		cart.state.static.dataRAM.data[dataAddr] = data
		cart.state.registers.Fetcher[f].inc()

	default:
		if poke {
			cart.banks[cart.state.bank][addr] = data
		}
	}

	return nil
}

// bankswitch on hotspot access.
func (cart *dpcPlus) bankswitch(addr uint16) bool {
	if addr >= 0x0ff6 && addr <= 0x0ffb {
		if addr == 0x0ff6 {
			cart.state.bank = 0
		} else if addr == 0x0ff7 {
			cart.state.bank = 1
		} else if addr == 0x0ff8 {
			cart.state.bank = 2
		} else if addr == 0x0ff9 {
			cart.state.bank = 3
		} else if addr == 0x0ffa {
			cart.state.bank = 4
		} else if addr == 0x0ffb {
			cart.state.bank = 5
		}
		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *dpcPlus) NumBanks() int {
	return len(cart.banks)
}

// GetBank implements the mapper.CartMapper interface.
func (cart *dpcPlus) GetBank(addr uint16) mapper.BankInfo {
	return mapper.BankInfo{
		Number:                cart.state.bank,
		IsRAM:                 false,
		ExecutingCoprocessor:  cart.state.callfn.IsActive(),
		CoprocessorResumeAddr: cart.state.callfn.ResumeAddr,
	}
}

// SetBank implements the mapper.CartMapper interface.
func (cart *dpcPlus) SetBank(bank string) error {
	if mapper.IsAutoBankSelection(bank) {
		cart.state.bank = len(cart.banks) - 1
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

// AccessPassive implements the mapper.CartMapper interface.
func (cart *dpcPlus) AccessPassive(addr uint16, data uint8) error {
	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *dpcPlus) Step(clock float32) {
	// sample rate of 20KHz.
	//
	// if Step() is called at a rate of 1.19Mhz. so:
	//
	// 1.19Mhz / 20KHz
	// = 59
	//
	// ie. we clock the music data fetchers once every 59 calls to Step() when
	// the VCS clock is running at 1.19Mhz
	//
	// the 20Khz is the same as the DPC format (see mapper_dpc for commentary).
	divisor := int(clock * 50)

	cart.state.beats++
	if cart.state.beats%divisor == 0 {
		cart.state.beats = 0
		cart.state.registers.MusicFetcher[0].Count += cart.state.registers.MusicFetcher[0].Freq
		cart.state.registers.MusicFetcher[1].Count += cart.state.registers.MusicFetcher[1].Freq
		cart.state.registers.MusicFetcher[2].Count += cart.state.registers.MusicFetcher[2].Freq
	}

	// Step ARM state if the ARM program is NOT running
	if cart.state.callfn.IsActive() {
		if cart.arm.ImmediateMode() {
			cart.arm.Step(clock)
		} else {
			r := cart.state.callfn.Step(clock, cart.arm.Clk)
			if r > 0 {
				cart.arm.Step(r)
			}
		}

		if !cart.state.callfn.IsActive() {
			cart.arm.ProcessProfiling()
		}
	} else {
		cart.arm.Step(clock)
	}
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *dpcPlus) CopyBanks() []mapper.BankContent {
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
func (cart *dpcPlus) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
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
func (cart *dpcPlus) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
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
		0x105a: {Symbol: "CALLFN", Action: mapper.HotspotFunction},
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
		0x1078: {Symbol: "DF0/queue", Action: mapper.HotspotRegister}, // DF0Write
		0x1079: {Symbol: "DF1/queue", Action: mapper.HotspotRegister},
		0x107a: {Symbol: "DF2/queue", Action: mapper.HotspotRegister},
		0x107b: {Symbol: "DF3/queue", Action: mapper.HotspotRegister},
		0x107c: {Symbol: "DF4/queue", Action: mapper.HotspotRegister},
		0x107d: {Symbol: "DF5/queue", Action: mapper.HotspotRegister},
		0x107e: {Symbol: "DF6/queue", Action: mapper.HotspotRegister},
		0x107f: {Symbol: "DF7/queue", Action: mapper.HotspotRegister},
	}
}

// ARMinterrupt implements the arm7tmdi.CatridgeHook interface.
func (cart *dpcPlus) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}

// CoProcExecutionState implements the coprocessor.CartCoProcBus interface.
func (cart *dpcPlus) CoProcExecutionState() coprocessor.CoProcExecutionState {
	if cart.state.callfn.IsActive() {
		return coprocessor.CoProcExecutionState{
			Sync:  coprocessor.CoProcNOPFeed,
			Yield: cart.state.yield,
		}
	}
	return coprocessor.CoProcExecutionState{
		Sync:  coprocessor.CoProcIdle,
		Yield: cart.state.yield,
	}
}

// CoProcRegister implements the coprocessor.CartCoProcBus interface.
func (cart *dpcPlus) GetCoProc() coprocessor.CartCoProc {
	return cart.arm
}

// SetYieldHook implements the coprocessor.CartCoProcBus interface.
func (cart *dpcPlus) SetYieldHook(hook coprocessor.CartYieldHook) {
	cart.yieldHook = hook
}

func (cart *dpcPlus) runArm() coprocessor.CoProcYield {
	yld, cycles := cart.arm.Run()
	if cycles > 0 || cart.env.Prefs.ARM.ImmediateCorrection.Get().(bool) {
		cart.state.callfn.Accumulate(cycles)
	}
	return yld
}
