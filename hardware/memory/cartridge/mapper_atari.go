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
	"math/bits"
	"os"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// from bankswitch_sizes.txt:
//
// 2K:
//
// -These carts are not bankswitched, however the data repeats twice in the
// 4K address space.  You'll need to manually double-up these images to 4K
// if you want to put these in say, a 4K cart.
//
// 4K:
//
// -These images are not bankswitched.
//
// 8K:
//
// -F8: This is the 'standard' method to implement 8K carts.  There are two
// addresses which select between two unique 4K sections.  They are 1FF8
// and 1FF9.  Any access to either one of these locations switches banks.
// Accessing 1FF8 switches in the first 4K, and accessing 1FF9 switches in
// the last 4K.  Note that you can only access one 4K at a time!
//
// 16K:
//
// -F6: The 'standard' method for implementing 16K of data.  It is identical
// to the F8 method above, except there are 4 4K banks.  You select which
// 4K bank by accessing 1FF6, 1FF7, 1FF8, and 1FF9.
//
// 32K:
//
// -F4: The 'standard' method for implementing 32K.  Only one cart is known
// to use it- Fatal Run.  Like the F6 method, however there are 8 4K
// banks instead of 4.  You use 1FF4 to 1FFB to select the desired bank.
//
//
// Some carts have extra RAM. The way of doing this for Atari format cartridges
// is with the addition of a "superchip".
//
// Atari's 'Super Chip' is nothing more than a 128-byte RAM chip that maps
// itsself in the first 256 bytes of cart memory.  (1000-10FFh) The first 128
// bytes is the write port, while the second 128 bytes is the read port. The
// difference in addresses is because there is no dedicated address line to the
// cart to differentiate between read and write operations.
//
// Some people have experimented with 1k carts. These are treated like 2k carts.

type atari struct {
	instance *instance.Instance

	mappingID string

	// atari formats apart from 2k and 4k are divided into banks. 2k and 4k
	// ROMs conceptually have one bank
	bankSize int
	banks    [][]uint8

	// rewindable state
	state *atariState

	// binary file has empty area(s). we use this to decide whether
	// the cartridge RAM should be allocated
	//
	// the superchip is not added automatically
	needsSuperchip bool
}

// size of superchip RAM
const superchipRAMsize = 128

// look for empty area (representing RAM) in binary data.
func hasEmptyArea(d []uint8) bool {
	// if the first byte in the cartridge is repeated 'superchipRAMsize' times
	// then we deem it to be an empty area.
	//
	// for example: the Fatal Run (NTSC) ROM uses FF rather than 00 to fill the
	// empty space.
	b := d[0]
	for i := 1; i < superchipRAMsize; i++ {
		if d[i] != b {
			return false
		}
	}
	return true
}

// ROMDump implements the mapper.CartROMDump interface.
func (cart *atari) ROMDump(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return curated.Errorf("atari: %v", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf("atari", err.Error())
		}
	}()

	for _, b := range cart.banks {
		_, err := f.Write(b)
		if err != nil {
			return curated.Errorf("atari: %v", err)
		}
	}

	return nil
}

// MappedBanks implements the mapper.CartMapper interface.
func (cart *atari) MappedBanks() string {
	return fmt.Sprintf("Bank: %d", cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart *atari) ID() string {
	return cart.mappingID
}

// Reset implements the mapper.CartMapper interface.
func (cart *atari) Reset() {
	for i := range cart.state.ram {
		if cart.instance.Prefs.RandomState.Get().(bool) {
			cart.state.ram[i] = uint8(cart.instance.Random.NoRewind(0xff))
		} else {
			cart.state.ram[i] = 0
		}
	}

	// known start-bank sensitive ROMs:
	//	(F8) Hack'Em Hangly Pacman which requires a start bank of 1
	//	(F8) CongoBongo which requires a start bank of 1
	//	(F6) Stay Frosty which cannot start in the last bank
	//	(F4) Strat-O-Gems which cannot start in bank 1
	switch cart.mappingID {
	case "2k": // (1 bank)
		cart.state.bank = 0
	case "4k": // (1 bank)
		cart.state.bank = 0
	case "F8": // 8k (2 banks)
		cart.state.bank = 1
	case "F6": // 16k (4 banks)
		cart.state.bank = 1
	case "F4": // 32k (8 banks)
		cart.state.bank = 0
	}
}

// GetBank implements the mapper.CartMapper interface.
func (cart *atari) GetBank(addr uint16) mapper.BankInfo {
	// because atari bank switching swaps out the entire memory space, every
	// address points to whatever the current bank is. compare to parker bros.
	// cartridges.
	return mapper.BankInfo{Number: cart.state.bank, IsRAM: cart.state.ram != nil && addr >= 0x80 && addr <= 0xff}
}

// Read implements the mapper.CartMapper interface.
func (cart *atari) Read(addr uint16, _ bool) (uint8, bool) {
	if cart.state.ram != nil {
		if addr >= 0x80 && addr <= 0xff {
			return cart.state.ram[addr-superchipRAMsize], true
		}
	}
	return 0, false
}

// Write implements the mapper.CartMapper interface.
func (cart *atari) Write(addr uint16, data uint8, poke bool) error {
	if cart.state.ram != nil {
		if addr <= 0x7f {
			cart.state.ram[addr] = data
			return nil
		}
	}

	if poke {
		cart.banks[cart.state.bank][addr] = data
		return nil
	}

	return curated.Errorf("atari: %v", curated.Errorf(cpubus.AddressError, addr))
}

// Patch implements the mapper.CartMapper interface.
func (cart *atari) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks) {
		return curated.Errorf("atari: %v", fmt.Errorf("patch offset too high (%v)", offset))
	}

	bank := offset / cart.bankSize
	offset %= cart.bankSize
	cart.banks[bank][offset] = data
	return nil
}

// Listen implements the mapper.CartMapper interface.
func (cart *atari) Listen(addr uint16, data uint8) {
	// Sometimes, cartridge addresses can be accessed inadvertently. in most
	// instances, there are no consequences but in the case of the Superchip,
	// the write addresses can be accessed and the RAM data changed. we handle
	// that here.
	//
	// https://atariage.com/forums/topic/329888-indexed-read-page-crossing-and-sc-ram/
	if cart.state.ram != nil {
		if addr&memorymap.OriginCart == memorymap.OriginCart {
			addr &= memorymap.MaskCart
			addr ^= memorymap.OriginCart
			// Atari Superchip is 128 bytes
			if addr&0xff80 == 0x0000 {
				cart.state.ram[addr&0x7f] = data
			}
		}
	}
}

// Step implements the mapper.CartMapper interface.
func (cart *atari) Step(_ float32) {
}

// GetRAM implements the mapper.CartRAMBus interface.
func (cart *atari) GetRAM() []mapper.CartRAM {
	if cart.state.ram == nil {
		return nil
	}

	r := make([]mapper.CartRAM, 1)
	r[0] = mapper.CartRAM{
		Label:  "Superchip",
		Origin: 0x1080,
		Data:   make([]uint8, len(cart.state.ram)),
		Mapped: true,
	}

	copy(r[0].Data, cart.state.ram)
	return r
}

// PutRAM implements the mapper.CartRAMBus interface.
func (cart *atari) PutRAM(_ int, idx int, data uint8) {
	if cart.state.ram == nil {
		return
	}

	cart.state.ram[idx] = data
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *atari) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))
	for b := 0; b < len(cart.banks); b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}
	return c
}

// AddSuperchip implements the mapper.OptionalSuperchip interface.
func (cart *atari) AddSuperchip(force bool) {
	if force || cart.needsSuperchip {
		cart.mappingID = fmt.Sprintf("%s (SC)", cart.mappingID)
		cart.state.ram = make([]uint8, superchipRAMsize)
	}
}

// atari4k is the original and most straightforward format:
//  - Pitfall
//  - River Raid
//  - Barnstormer
//  - etc.
type atari4k struct {
	atari
}

func newAtari4k(instance *instance.Instance, data []byte) (mapper.CartMapper, error) {
	cart := &atari4k{}
	cart.instance = instance
	cart.bankSize = 4096
	cart.mappingID = "4k"
	cart.banks = make([][]uint8, 1)
	cart.needsSuperchip = hasEmptyArea(data)
	cart.state = newAtariState()

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, curated.Errorf("4k: %v", "wrong number of bytes in the cartridge data")
	}

	cart.banks[0] = make([]uint8, cart.bankSize)
	copy(cart.banks[0], data)

	return cart, nil
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *atari4k) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *atari4k) Plumb() {
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *atari4k) NumBanks() int {
	return 1
}

// Read implements the mapper.CartMapper interface.
func (cart *atari4k) Read(addr uint16, peek bool) (uint8, error) {
	if data, ok := cart.atari.Read(addr, peek); ok {
		return data, nil
	}
	return cart.banks[0][addr], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *atari4k) Write(addr uint16, data uint8, poke bool) error {
	return cart.atari.Write(addr, data, poke)
}

// atari2k is the half-size cartridge of 2048 bytes:
//	- Combat
//  - Dragster
//  - Outlaw
//	- Surround
//  - early cartridges
//
// it also supports other cartridge sizes less than 4096 bytes
type atari2k struct {
	atari
	mask uint16
}

func newAtari2k(instance *instance.Instance, data []byte) (mapper.CartMapper, error) {
	sz := len(data)

	// support any size less than 4096 bytes that is a power of two
	if sz >= 4096 || bits.OnesCount(uint(sz)) != 1 {
		return nil, curated.Errorf("unsupported cartridge size")
	}

	cart := &atari2k{}
	cart.instance = instance
	cart.bankSize = sz
	cart.mappingID = "2k"
	cart.banks = make([][]uint8, 1)
	cart.needsSuperchip = hasEmptyArea(data)
	cart.state = newAtariState()
	cart.mask = uint16(sz - 1)

	cart.banks[0] = make([]uint8, cart.bankSize)
	copy(cart.banks[0], data)

	return cart, nil
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *atari2k) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *atari2k) Plumb() {
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *atari2k) NumBanks() int {
	return 1
}

// Read implements the mapper.CartMapper interface.
func (cart *atari2k) Read(addr uint16, peek bool) (uint8, error) {
	if data, ok := cart.atari.Read(addr, peek); ok {
		return data, nil
	}
	return cart.banks[0][addr&cart.mask], nil
}

// IterateBank implements the mapper.CartMapper interface.
func (cart *atari2k) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, 1)
	c[0] = mapper.BankContent{Number: 0,
		Data:    cart.banks[0],
		Origins: []uint16{memorymap.OriginCart, memorymap.OriginCart + uint16(cart.bankSize)},
	}
	return c
}

// Write implements the mapper.CartMapper interface.
func (cart *atari2k) Write(addr uint16, data uint8, poke bool) error {
	return cart.atari.Write(addr, data, poke)
}

// atari8k (F8):
//	- ET
//  - Krull
//  - etc.
type atari8k struct {
	atari
}

func newAtari8k(instance *instance.Instance, data []uint8) (mapper.CartMapper, error) {
	cart := &atari8k{}
	cart.instance = instance
	cart.bankSize = 4096
	cart.mappingID = "F8"
	cart.banks = make([][]uint8, cart.NumBanks())
	cart.needsSuperchip = hasEmptyArea(data)
	cart.state = newAtariState()

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, curated.Errorf("F8: %v", "wrong number of bytes in the cartridge data")
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *atari8k) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *atari8k) Plumb() {
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *atari8k) NumBanks() int {
	return 2
}

// Read implements the mapper.CartMapper interface.
func (cart *atari8k) Read(addr uint16, peek bool) (uint8, error) {
	if data, ok := cart.atari.Read(addr, peek); ok {
		return data, nil
	}

	if !peek {
		cart.bankswitch(addr)
	}

	return cart.banks[cart.state.bank][addr], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *atari8k) Write(addr uint16, data uint8, poke bool) error {
	if !poke {
		if cart.bankswitch(addr) {
			return nil
		}
	}

	return cart.atari.Write(addr, data, poke)
}

// bankswitch on hotspot access.
func (cart *atari8k) bankswitch(addr uint16) bool {
	if addr >= 0x0ff8 && addr <= 0x0ff9 {
		if addr == 0x0ff8 {
			cart.state.bank = 0
		} else if addr == 0x0ff9 {
			cart.state.bank = 1
		}
		return true
	}
	return false
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart *atari8k) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff8: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *atari8k) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}

// atari16k (F6):
//	- Crystal Castle
//	- RS Boxing
//  - Midnite Magic
//  - etc.
type atari16k struct {
	atari
}

func newAtari16k(instance *instance.Instance, data []byte) (mapper.CartMapper, error) {
	cart := &atari16k{}
	cart.instance = instance
	cart.bankSize = 4096
	cart.mappingID = "F6"
	cart.banks = make([][]uint8, cart.NumBanks())
	cart.needsSuperchip = hasEmptyArea(data)
	cart.state = newAtariState()

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, curated.Errorf("F6: %v", "wrong number of bytes in the cartridge data")
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *atari16k) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *atari16k) Plumb() {
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *atari16k) NumBanks() int {
	return 4
}

// Read implements the mapper.CartMapper interface.
func (cart *atari16k) Read(addr uint16, peek bool) (uint8, error) {
	if data, ok := cart.atari.Read(addr, peek); ok {
		return data, nil
	}

	if !peek {
		cart.bankswitch(addr)
	}

	return cart.banks[cart.state.bank][addr], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *atari16k) Write(addr uint16, data uint8, poke bool) error {
	if !poke {
		if cart.bankswitch(addr) {
			return nil
		}
	}

	return cart.atari.Write(addr, data, poke)
}

// bankswitch on hotspot access.
func (cart *atari16k) bankswitch(addr uint16) bool {
	if addr >= 0x0ff6 && addr <= 0x0ff9 {
		if addr == 0x0ff6 {
			cart.state.bank = 0
		} else if addr == 0x0ff7 {
			cart.state.bank = 1
		} else if addr == 0x0ff8 {
			cart.state.bank = 2
		} else if addr == 0x0ff9 {
			cart.state.bank = 3
		}
		return true
	}
	return false
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart *atari16k) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff6: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff7: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1ff8: {Symbol: "BANK2", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK3", Action: mapper.HotspotBankSwitch},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart *atari16k) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}

// atari32k (F8)
// - Fatal Run
// - Super Mario Bros.
// - Donkey Kong (homebrew)
// - etc.
type atari32k struct {
	atari
}

func newAtari32k(instance *instance.Instance, data []byte) (mapper.CartMapper, error) {
	cart := &atari32k{}
	cart.instance = instance
	cart.bankSize = 4096
	cart.mappingID = "F4"
	cart.banks = make([][]uint8, cart.NumBanks())
	cart.needsSuperchip = hasEmptyArea(data)
	cart.state = newAtariState()

	if len(data) != cart.bankSize*cart.NumBanks() {
		return nil, curated.Errorf("F4: %v", "wrong number of bytes in the cartridge data")
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		copy(cart.banks[k], data[offset:offset+cart.bankSize])
	}

	return cart, nil
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *atari32k) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *atari32k) Plumb() {
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *atari32k) NumBanks() int {
	return 8
}

// Read implements the mapper.CartMapper interface.
func (cart *atari32k) Read(addr uint16, peek bool) (uint8, error) {
	if data, ok := cart.atari.Read(addr, peek); ok {
		return data, nil
	}

	if !peek {
		cart.bankswitch(addr)
	}

	return cart.banks[cart.state.bank][addr], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *atari32k) Write(addr uint16, data uint8, poke bool) error {

	if !poke {
		if cart.bankswitch(addr) {
			return nil
		}
	}

	return cart.atari.Write(addr, data, poke)
}

// bankswitch on hotspot access.
func (cart *atari32k) bankswitch(addr uint16) bool {
	if addr >= 0x0ff4 && addr <= 0xffb {
		if addr == 0x0ff4 {
			cart.state.bank = 0
		} else if addr == 0x0ff5 {
			cart.state.bank = 1
		} else if addr == 0x0ff6 {
			cart.state.bank = 2
		} else if addr == 0x0ff7 {
			cart.state.bank = 3
		} else if addr == 0x0ff8 {
			cart.state.bank = 4
		} else if addr == 0x0ff9 {
			cart.state.bank = 5
		} else if addr == 0x0ffa {
			cart.state.bank = 6
		} else if addr == 0x0ffb {
			cart.state.bank = 7
		}
		return true
	}
	return false
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart *atari32k) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
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
func (cart *atari32k) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return cart.ReadHotspots()
}

// rewindable state for all atari cartridges.
type atariState struct {
	// identifies the currently selected bank
	bank int

	// some atari ROMs support aditional RAM. this is sometimes referred to as
	// the superchip. ram is only added when it is detected
	ram []uint8
}

func newAtariState() *atariState {
	// not allocating memory for RAM because some atari carts don't have it.
	// memory allocated in addSuperChip() if required
	return &atariState{}
}

// Snapshot implements the mapper.CartMapper interface.
func (s *atariState) Snapshot() *atariState {
	n := *s
	if s.ram != nil {
		n.ram = make([]uint8, len(s.ram))
		copy(n.ram, s.ram)
	}
	return &n
}
