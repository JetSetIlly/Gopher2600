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
	"debug/dwarf"
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// cdf implements the mapper.CartMapper interface.
type cdf struct {
	instance *instance.Instance
	dev      mapper.CartCoProcDeveloper

	pathToROM string
	mappingID string

	// additional CPU - used by some ROMs
	arm *arm.ARM

	// cdf comes in several different versions
	version version

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

// registers should be accessed via readDatastreamPointer() and
// updateDatastreamPointer(). Actually reading the data in the data stream
// should be done by streamData().
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

// NewCDF is the preferred method of initialisation for the CDF type.
func NewCDF(instance *instance.Instance, pathToROM string, version string, data []byte) (mapper.CartMapper, error) {
	cart := &cdf{
		instance:  instance,
		pathToROM: pathToROM,
		mappingID: "CDF",
		bankSize:  4096,
		state:     newCDFstate(),
	}

	var err error
	cart.version, err = newVersion(instance.Prefs.ARM.Model.Get().(string), version, data)
	if err != nil {
		return nil, curated.Errorf("CDF: %v", err)
	}

	// allocate enough banks
	cart.banks = make([][]uint8, cart.NumBanks())

	// partition data into banks
	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		offset += driverSize
		if cart.version.submapping != "CDFJ+" {
			offset += customSize
		}
		cart.banks[k] = data[offset : offset+cart.bankSize]
	}

	// initialise static memory
	cart.state.static = cart.newCDFstatic(instance, data, cart.version)

	// datastream registers need to reference the incrementShift and
	// fetcherShift values in the version type. we make a copy of these values
	// on ROM initialisation
	for i := range cart.state.registers.Datastream {
		cart.state.registers.Datastream[i].incrementShift = cart.version.incrementShift
		cart.state.registers.Datastream[i].fetcherShift = cart.version.fetcherShift
	}

	// initialise ARM processor
	//
	// if bank0 has any ARM code then it will start at offset 0x08. first eight
	// bytes are the ARM header
	cart.arm = arm.NewARM(cart.version.arch, cart.version.mamcr, cart.version.mmap, cart.instance.Prefs.ARM, cart.state.static, cart, cart.pathToROM)

	return cart, nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *cdf) MappedBanks() string {
	return fmt.Sprintf("Bank: %d", cart.state.bank)
}

// CoProcID implements the mapper.CartCoProc interface.
func (cart *cdf) CoProcID() string {
	return cart.arm.CoProcID()
}

// SetDisassembler implements the mapper.CartCoProc interface.
func (cart *cdf) SetDisassembler(disasm mapper.CartCoProcDisassembler) {
	cart.arm.SetDisassembler(disasm)
}

// SetDeveloper implements the mapper.CartCoProc interface.
func (cart *cdf) SetDeveloper(dev mapper.CartCoProcDeveloper) {
	cart.dev = dev
	cart.arm.SetDeveloper(dev)
}

// ID implements the mapper.CartMapper interface.
func (cart *cdf) ID() string {
	return cart.version.submapping
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *cdf) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *cdf) Plumb() {
	cart.arm.Plumb(nil, cart.state.static, cart)
}

// Plumb implements the mapper.CartMapper interface.
func (cart *cdf) PlumbFromDifferentEmulation() {
	cart.arm = arm.NewARM(cart.version.arch, cart.version.mamcr, cart.version.mmap, cart.instance.Prefs.ARM, cart.state.static, cart, cart.pathToROM)
}

// Reset implements the mapper.CartMapper interface.
func (cart *cdf) Reset() {
	bank := len(cart.banks) - 1
	if cart.version.submapping == "CDFJ+" {
		bank = 0
	}
	cart.state.initialise(bank)
}

const (
	jmpAbsolute  = 0x4c
	ldaImmediate = 0xa9
	ldxImmediate = 0xa2
	ldyImmediate = 0xa0
)

// Read implements the mapper.CartMapper interface.
func (cart *cdf) Read(addr uint16, passive bool) (uint8, error) {
	if b, ok := cart.state.callfn.Check(addr, cart.arm.BreakpointHasTriggered()); ok {
		return b, nil
	}

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
			jmp := cart.readDatastreamPointer(reg)
			data = cart.state.static.dataRAM[jmp>>cart.version.fetcherShift]
			jmp += 1 << cart.version.fetcherShift
			cart.updateDatastreamPointer(reg, jmp)

			return data, nil
		}
	}

	// any fastjmp preparation that wasn't serviced by the above branch must be
	// a false positive, by definition.
	cart.state.fastJMP = 0

	if cart.state.registers.FastFetch && cart.state.fastLoad {
		cart.state.fastLoad = false

		// data fetchers
		if data >= cart.version.datastreamOffset && data <= cart.version.datastreamOffset+DSCOMM {
			return cart.streamData(int(data - cart.version.datastreamOffset)), nil
		}

		// music fetchers
		if data == byte(cart.version.amplitudeRegister) {
			if cart.state.registers.SampleMode {
				addr := cart.readMusicFetcher(0)
				addr += cart.state.registers.MusicFetcher[0].Count >> (cart.version.musicFetcherShift + 1)

				// get sample from memory
				data, _ = cart.state.static.Read8bit(addr)

				// prevent excessive volume
				if cart.state.registers.MusicFetcher[0].Count&(1<<cart.version.musicFetcherShift) == 0 {
					data >>= 4
				}

				return data, nil
			}

			// data retrieval for non-SampleMode uses all three music fetchers
			data = 0
			for i := range cart.state.registers.MusicFetcher {
				m := cart.readMusicFetcher(i)
				m += (cart.state.registers.MusicFetcher[i].Count >> cart.state.registers.MusicFetcher[i].Waveform)
				v, _ := cart.state.static.Read8bit(m)
				data += v
			}

			return data, nil
		}

		// if data is higher than AMPLITUDE then the 0xa9 we detected in the
		// previous Read() was just a normal value (maybe an LDA #immediate
		// opcode but not one intended for fast fetch)
	}

	// set lda flag if fast fetch mode is on and data returned is LDA #immediate
	switch data {
	case ldaImmediate:
		cart.state.fastLoad = cart.state.registers.FastFetch
	case ldxImmediate:
		cart.state.fastLoad = cart.state.registers.FastFetch && cart.version.fastLDX
	case ldyImmediate:
		cart.state.fastLoad = cart.state.registers.FastFetch && cart.version.fastLDY
	}

	// set jmp flag if fast fetch mode is on and data returned is JMP absolute
	if cart.state.registers.FastFetch && data == jmpAbsolute {
		// only "jmp absolute" instructions with certain address operands are
		// treated as "FastJMPs". Generally, this address must be $0000 but in
		// the case of the CDFJ version an address of $0100 is also acceptable.
		if cart.banks[cart.state.bank][addr+1]&cart.version.fastJMPmask == 0x00 && cart.banks[cart.state.bank][addr+2] == 0x00 {
			// JMP operator takes a 16bit operand so we'll count the number of
			// bytes we've read
			cart.state.fastJMP = 2
		}
	}

	return data, nil
}

// Write implements the mapper.CartMapper interface.
func (cart *cdf) Write(addr uint16, data uint8, passive bool, poke bool) error {
	// bank switches can not take place if coprocessor is active
	if cart.state.callfn.IsActive() {
		return nil
	}

	if cart.bankswitch(addr, passive) {
		return nil
	}

	switch addr {
	case 0x0ff0:
		// DSWRITE

		// top 12 bits are significant
		v := cart.readDatastreamPointer(DSCOMM)

		// write data to ARM RAM
		cart.state.static.dataRAM[v>>cart.version.fetcherShift] = data

		// advance address value
		v += 1 << cart.version.fetcherShift

		// write adjusted address (making sure to put the bits in the top 12 bits)
		cart.updateDatastreamPointer(DSCOMM, v)

	case 0x0ff1:
		// DSPTR
		v := cart.readDatastreamPointer(DSCOMM) << 8
		v &= cart.version.fetcherMask

		// add new data to lower byte of dsptr value
		v |= (uint32(data) << cart.version.fetcherShift)

		// write dsptr to dscomm register
		cart.updateDatastreamPointer(DSCOMM, v)

	case 0x0ff2:
		// SETMODE
		cart.state.registers.FastFetch = data&0x0f != 0x0f
		cart.state.registers.SampleMode = data&0xf0 != 0xf0

		if !cart.state.registers.FastFetch {
			cart.state.fastLoad = false
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
			if cart.dev != nil {
				cart.dev.StartProfiling()
			}
			err := cart.runArm()
			if err != nil {
				return err
			}
		}

	default:
		if poke {
			cart.banks[cart.state.bank][addr] = data
			return nil
		}

		return curated.Errorf("CDF: %v", curated.Errorf(cpubus.AddressError, addr))
	}

	return nil
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

		if cart.version.submapping == "CDFJ+" {
			cart.state.bank++
			cart.state.bank %= 7
		}

		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *cdf) NumBanks() int {
	return 7
}

// GetBank implements the mapper.CartMapper interface.
func (cart *cdf) GetBank(addr uint16) mapper.BankInfo {
	return mapper.BankInfo{
		Number:                cart.state.bank,
		IsRAM:                 false,
		ExecutingCoprocessor:  cart.state.callfn.IsActive(),
		CoprocessorResumeAddr: cart.state.callfn.ResumeAddr,
	}
}

// Patch implements the mapper.CartMapper interface.
func (cart *cdf) Patch(offset int, data uint8) error {
	return curated.Errorf("CDF: patching unsupported")
}

// Listen implements the mapper.CartMapper interface.
func (cart *cdf) Listen(addr uint16, data uint8) {
}

// Step implements the mapper.CartMapper interface.
func (cart *cdf) Step(clock float32) {
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

	// Step ARM state if the ARM program is NOT running
	if cart.state.callfn.IsActive() {
		if cart.state.immediateMode {
			cart.arm.Step(clock)
		} else {
			timerClock := cart.state.callfn.Step(clock, cart.arm.Clk)
			if timerClock > 0 {
				cart.arm.Step(timerClock)
			}
		}

		if cart.dev != nil && !cart.state.callfn.IsActive() {
			cart.dev.ProcessProfiling()
		}
	} else {
		cart.arm.Step(clock)
	}
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

// ReadHotspots implements the mapper.CartLabelsBus interface.
func (cart *cdf) Labels() mapper.CartLabels {
	return map[uint16]string{
		0x0000: "FASTJMP1",
		0x0001: "FASTJMP2",
	}
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
		0x1ff5: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff6: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1ff7: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1ff8: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK4", Action: mapper.HotspotBankSwitch},
		0x1ffa: {Symbol: "BANK5", Action: mapper.HotspotBankSwitch},
		0x1ffb: {Symbol: "BANK6", Action: mapper.HotspotBankSwitch},
	}
}

// ARMinterrupt implements the arm.CatridgeHook interface.
func (cart *cdf) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	var r arm.ARMinterruptReturn

	if cart.version.submapping == "CDF0" {
		switch addr {
		case cart.version.mmap.FlashOrigin | 0x000006e2:
			r.InterruptEvent = "Set music note"
			if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
				return r, curated.Errorf("music fetcher index (%d) too high ", val1)
			}
			cart.state.registers.MusicFetcher[val1].Freq = val2
			r.NumMemAccess = 2
			r.NumAdditionalCycles = 11
		case cart.version.mmap.FlashOrigin | 0x000006e6:
			// reset wave
			r.InterruptEvent = "Reset wave"
			if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
				return r, curated.Errorf("music fetcher index (%d) too high ", val1)
			}
			cart.state.registers.MusicFetcher[val1].Count = 0
			r.NumMemAccess = 3
			r.NumAdditionalCycles = 13
		case cart.version.mmap.FlashOrigin | 0x000006ea:
			r.InterruptEvent = "Get wave pointer"
			if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
				return r, curated.Errorf("music fetcher index (%d) too high ", val1)
			}
			r.SaveValue = cart.state.registers.MusicFetcher[val1].Count
			r.SaveRegister = 2
			r.SaveResult = true
			r.NumMemAccess = 3
			r.NumAdditionalCycles = 13
		case cart.version.mmap.FlashOrigin | 0x000006ee:
			r.InterruptEvent = "Set wave size"
			if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
				return r, curated.Errorf("music fetcher index (%d) too high ", val1)
			}
			cart.state.registers.MusicFetcher[val1].Waveform = uint8(val2)
			r.NumMemAccess = 3
			r.NumAdditionalCycles = 28
		default:
			return r, nil
		}

		r.InterruptServiced = true
		return r, nil
	}

	switch addr {
	case cart.version.mmap.FlashOrigin | 0x00000752:
		r.InterruptEvent = "Set music note"
		if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
			return r, curated.Errorf("music fetcher index (%d) too high ", val1)
		}
		cart.state.registers.MusicFetcher[val1].Freq = val2
		r.NumMemAccess = 2
		r.NumAdditionalCycles = 11
	case cart.version.mmap.FlashOrigin | 0x00000756:
		r.InterruptEvent = "Reset wave"
		if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
			return r, curated.Errorf("music fetcher index (%d) too high ", val1)
		}
		cart.state.registers.MusicFetcher[val1].Count = 0
		r.NumMemAccess = 3
		r.NumAdditionalCycles = 13
	case cart.version.mmap.FlashOrigin | 0x0000075a:
		r.InterruptEvent = "Get wave pointer"
		if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
			return r, curated.Errorf("music fetcher index (%d) too high ", val1)
		}
		r.SaveValue = cart.state.registers.MusicFetcher[val1].Count
		r.SaveRegister = 2
		r.SaveResult = true
		r.NumMemAccess = 3
		r.NumAdditionalCycles = 13
	case cart.version.mmap.FlashOrigin | 0x0000075e:
		r.InterruptEvent = "Set wave size"
		if val1 >= uint32(len(cart.state.registers.MusicFetcher)) {
			return r, curated.Errorf("music fetcher index (%d) too high ", val1)
		}
		cart.state.registers.MusicFetcher[val1].Waveform = uint8(val2)
		r.NumMemAccess = 3
		r.NumAdditionalCycles = 28
	default:
		return r, nil
	}

	r.InterruptServiced = true
	return r, nil
}

// HotLoad implements the mapper.CartHotLoader interface.
func (cart *cdf) HotLoad(data []byte) error {
	if len(data) == 0 {
		return curated.Errorf("CDF: empty data")
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		offset += driverSize + customSize
		cart.banks[k] = data[offset : offset+cart.bankSize]
	}

	cart.state.static.HotLoad(data)

	cart.arm.Plumb(nil, cart.state.static, cart)
	cart.arm.ClearCaches()

	return nil
}

// CoProcState implements the mapper.CartCoProc interface.
func (cart *cdf) CoProcState() mapper.CoProcState {
	if cart.state.callfn.IsActive() {
		return mapper.CoProcNOPFeed
	}
	return mapper.CoProcIdle
}

// BreakpointHasTriggered implements the mapper.CartCoProc interface.
func (cart *cdf) BreakpointHasTriggered() bool {
	return cart.arm.BreakpointHasTriggered()
}

// ResumeAfterBreakpoint implements the mapper.CartCoProc interface.
func (cart *cdf) ResumeAfterBreakpoint() error {
	if cart.arm.BreakpointHasTriggered() {
		return cart.runArm()
	}
	return nil
}

// BreakpointsDisable implements the mapper.CartCoProc interface.
func (cart *cdf) BreakpointsDisable(disable bool) {
	cart.arm.BreakpointsDisable(disable)
}

func (cart *cdf) runArm() error {
	cart.state.immediateMode = cart.instance.Prefs.ARM.Immediate.Get().(bool)

	cycles, err := cart.arm.Run()
	if err != nil {
		return curated.Errorf("CDF: %v", err)
	}

	cart.state.callfn.Start(cycles)

	// update the Register types after completion of each CALLFN
	for i := range cart.state.registers.Datastream {
		cart.state.registers.Datastream[i].Pointer = cart.readDatastreamPointer(i)
		cart.state.registers.Datastream[i].Increment = cart.readDatastreamIncrement(i)
		cart.state.registers.Datastream[i].AfterCALLFN = cart.readDatastreamPointer(i)
	}

	return nil
}

// DWARF implements the mapper.CartCoProc interface.
func (cart *cdf) DWARF() *dwarf.Data {
	return nil
}

// ELFSection implements the mapper.CartCoProc interface.
func (cart *cdf) ELFSection(name string) (uint32, bool) {
	return 0, false
}
