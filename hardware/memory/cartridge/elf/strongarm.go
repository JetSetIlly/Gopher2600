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

package elf

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// signature of a strongarm function
type strongArmFunction func()

// state of the strongarm emulation. not all ELF binaries make uses of the
// strongarm functions, in those instances strongArmState will be unused
type strongArmState struct {
	function  strongArmFunction
	state     int
	registers [arm.NumRegisters]uint32

	// the expected next 6507 address to be working with
	nextRomAddress uint16

	// the vcsCopyOverblankToRiotRam() function is a loop. we need to keep
	// track of the loop counter and sub-state in addition to the normal state
	// value
	//
	// the mechanism can be used for other looping functions
	counter  int
	subState int
}

// strongARM functions need to return to the main program with a branch exchange
var strongArmStub = []byte{
	0x70, 0x47, // BX LR
	0x00, 0x00,
}

// setStrongArmFunction initialises the next function to run. It takes a copy of the
// ARM registers at that point of initialisation
func (mem *elfMemory) setStrongArmFunction(f strongArmFunction) {
	mem.strongarm.function = f
	mem.strongarm.state = 0
	mem.strongarm.registers = mem.arm.Registers()
}

// a strongArmFunction should always end with a call to endFunction() no matter
// how many execution states it has.
func (mem *elfMemory) endStrongArmFunction() {
	mem.strongarm.function = nil
}

// memset works like you might expect but should only be called directly from
// the ARM emulation (and not from anothr function called by the ARM
// emulation). this is because this memset() function ends with a call to
// mem.endStrongArmFunction() which will kill the parent function too
func (mem *elfMemory) memset() {
	addr := mem.strongarm.registers[0]
	m, o := mem.MapAddress(addr, true)

	v := mem.strongarm.registers[1]
	l := mem.strongarm.registers[2]
	for i := uint32(0); i < l; i++ {
		(*m)[o+i] = byte(v)
	}

	mem.endStrongArmFunction()
}

func (mem *elfMemory) memcpy() {
	panic("memcpy")
}

func (mem *elfMemory) setNextRomAddress(addr uint16) {
	mem.strongarm.nextRomAddress = addr & memorymap.Memtop
}

func (mem *elfMemory) injectRomByte(v uint8) bool {
	addrIn := uint16(mem.gpio.A[toArm_address])
	addrIn |= uint16(mem.gpio.A[toArm_address+1]) << 8
	addrIn &= memorymap.Memtop

	if addrIn != mem.strongarm.nextRomAddress {
		return false
	}

	mem.gpio.B[fromArm_Opcode] = v
	mem.strongarm.nextRomAddress++

	return true
}

func (mem *elfMemory) yieldDataBus(addr uint16) bool {
	addrIn := uint16(mem.gpio.A[toArm_address])
	addrIn |= uint16(mem.gpio.A[toArm_address+1]) << 8
	addrIn &= memorymap.Memtop

	if addrIn != addr {
		return false
	}

	return true
}

// void vcsWrite3(uint8_t ZP, uint8_t data)
func (mem *elfMemory) vcsWrite3() {
	panic("vcsWrite3")
}

// void vcsJmp3()
func (mem *elfMemory) vcsJmp3() {
	switch mem.strongarm.state {
	case 0:
		if mem.injectRomByte(0x4c) {
			mem.strongarm.state++
		}
	case 1:
		if mem.injectRomByte(0x00) {
			mem.strongarm.state++
		}
	case 2:
		if mem.injectRomByte(0x10) {
			mem.endStrongArmFunction()
			mem.setNextRomAddress(0x1000)
		}
	}
}

// void vcsLda2(uint8_t data)
func (mem *elfMemory) vcsLda2() {
	data := uint8(mem.strongarm.registers[0])
	switch mem.strongarm.state {
	case 0:
		if mem.injectRomByte(0xa9) {
			mem.strongarm.state++
		}
	case 1:
		if mem.injectRomByte(data) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsSta3(uint8_t ZP)
func (mem *elfMemory) vcsSta3() {
	zp := uint8(mem.strongarm.registers[0])
	switch mem.strongarm.state {
	case 0:
		if mem.injectRomByte(0x85) {
			mem.strongarm.state++
		}
	case 1:
		if mem.injectRomByte(zp) {
			mem.strongarm.state++
		}
	case 2:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// uint8_t snoopDataBus(uint16_t address)
func (mem *elfMemory) snoopDataBus() {
	addrIn := uint16(mem.gpio.A[toArm_address])
	addrIn |= uint16(mem.gpio.A[toArm_address+1]) << 8
	addrIn &= memorymap.Memtop

	switch mem.strongarm.state {
	case 0:
		if addrIn == mem.strongarm.nextRomAddress {
			mem.strongarm.registers[0] = uint32(mem.gpio.B[toArm_data])
			mem.arm.SetRegisters(mem.strongarm.registers)
			mem.endStrongArmFunction()
		}
	}

	// note that this implementation of snoopDataBus is missing the "give
	// peripheral time to respond" loop that we see in the real vcsLib
}

// uint8_t vcsRead4(uint16_t address)
func (mem *elfMemory) vcsRead4() {
	address := uint16(mem.strongarm.registers[0])
	address &= memorymap.Memtop

	switch mem.strongarm.state {
	case 0:
		if mem.injectRomByte(0xad) {
			mem.strongarm.state++
		}
	case 1:
		if mem.injectRomByte(uint8(address)) {
			mem.strongarm.state++
		}
	case 2:
		if mem.injectRomByte(uint8(address >> 8)) {
			mem.setStrongArmFunction(mem.snoopDataBus)
			mem.strongarm.function()
		}
	}
}

// void vcsStartOverblank()
func (mem *elfMemory) vcsStartOverblank() {
	switch mem.strongarm.state {
	case 0:
		if mem.injectRomByte(0x4c) {
			mem.strongarm.state++
		}
	case 1:
		if mem.injectRomByte(0x80) {
			mem.strongarm.state++
		}
	case 2:
		if mem.injectRomByte(0x00) {
			mem.strongarm.state++
		}
	case 3:
		if mem.yieldDataBus(uint16(0x0080)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsEndOverblank()
func (mem *elfMemory) vcsEndOverblank() {
	switch mem.strongarm.state {
	case 0:
		mem.setNextRomAddress(0x1fff)
		if mem.injectRomByte(0x00) {
			mem.strongarm.state++
		}
	case 1:
		if mem.yieldDataBus(uint16(0x00ac)) {
			mem.setNextRomAddress(0x1000)
			mem.endStrongArmFunction()
		}
	}
}

// void vcsLdaForBusStuff2()
func (mem *elfMemory) vcsLdaForBusStuff2() {
	panic("vcsLdaForBusStuff2")
}

// void vcsLdxForBusStuff2()
func (mem *elfMemory) vcsLdxForBusStuff2() {
	panic("vcsLdxForBusStuff2")
}

// void vcsLdyForBusStuff2()
func (mem *elfMemory) vcsLdyForBusStuff2() {
	panic("vcsLdyForBusStuff2")
}

// void vcsWrite5(uint8_t ZP, uint8_t data)
func (mem *elfMemory) vcsWrite5() {
	zp := uint8(mem.strongarm.registers[0])
	data := uint8(mem.strongarm.registers[1])
	switch mem.strongarm.state {
	case 0:
		if mem.injectRomByte(0xa9) {
			mem.strongarm.state++
		}
	case 1:
		if mem.injectRomByte(data) {
			mem.strongarm.state++
		}
	case 2:
		if mem.injectRomByte(0x85) {
			mem.strongarm.state++
		}
	case 3:
		if mem.injectRomByte(zp) {
			mem.strongarm.state++
		}
	case 4:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsLdx2(uint8_t data)
func (mem *elfMemory) vcsLdx2() {
	panic("vcsLdx2")
}

// void vcsLdy2(uint8_t data)
func (mem *elfMemory) vcsLdy2() {
	panic("vcsLdy2")
}

// void vcsSta4(uint8_t ZP)
func (mem *elfMemory) vcsSta4() {
	panic("vcsSta4")
}

// void vcsStx3(uint8_t ZP)
func (mem *elfMemory) vcsStx3() {
	panic("vcsStx3")
}

// void vcsStx4(uint8_t ZP)
func (mem *elfMemory) vcsStx4() {
	panic("vcsStx4")
}

// void vcsSty3(uint8_t ZP)
func (mem *elfMemory) vcsSty3() {
	panic("vcsSty3")
}

// void vcsSty4(uint8_t ZP)
func (mem *elfMemory) vcsSty4() {
	panic("vcsSty4")
}

// void vcsTxs2()
func (mem *elfMemory) vcsTxs2() {
	panic("vcsTxs2")
}

// void vcsJsr6(uint16_t target)
func (mem *elfMemory) vcsJsr6() {
	switch mem.strongarm.state {
	case 0:
		if mem.injectRomByte(0x20) {
			mem.strongarm.state++
		}
	case 1:
		if mem.injectRomByte(uint8(mem.strongarm.registers[0])) {
			mem.strongarm.state++
		}
	case 2:
		if mem.injectRomByte(uint8(mem.strongarm.registers[0] >> 8)) {
			mem.gpio.A[toArm_address] = uint8(mem.strongarm.registers[0])
			mem.gpio.A[toArm_address+1] = uint8(mem.strongarm.registers[0] >> 8)
			mem.gpio.A[toArm_address+2] = uint8(mem.strongarm.registers[0] >> 16)
			mem.gpio.A[toArm_address+3] = uint8(mem.strongarm.registers[0] >> 24)

			mem.endStrongArmFunction()
		}
	}
}

// void vcsNop2()
func (mem *elfMemory) vcsNop2() {
	switch mem.strongarm.state {
	case 0:
		if mem.injectRomByte(0xea) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsNop2n(uint16_t n)
func (mem *elfMemory) vcsNop2n() {
	switch mem.strongarm.state {
	case 0:
		if mem.injectRomByte(0xea) {
			mem.strongarm.nextRomAddress += uint16(mem.strongarm.registers[0]) - 1
			mem.endStrongArmFunction()
		}
	}
}

// void vcsCopyOverblankToRiotRam()
func (mem *elfMemory) vcsCopyOverblankToRiotRam() {
	switch mem.strongarm.state {
	case 0:
		if mem.strongarm.counter >= len(overblank) {
			mem.endStrongArmFunction()
			return
		}
		mem.strongarm.state++
		mem.strongarm.subState = 0
		fallthrough
	case 1:
		switch mem.strongarm.subState {
		case 0:
			if mem.injectRomByte(0xa9) {
				mem.strongarm.subState++
			}
		case 1:
			if mem.injectRomByte(overblank[mem.strongarm.counter]) {
				mem.strongarm.subState++
			}
		case 2:
			if mem.injectRomByte(0x85) {
				mem.strongarm.subState++
			}
		case 3:
			if mem.injectRomByte(uint8(0x80 + mem.strongarm.counter)) {
				mem.strongarm.subState++
			}
		case 4:
			if mem.yieldDataBus(uint16(0x80 + mem.strongarm.counter)) {
				mem.strongarm.counter++
				mem.strongarm.state = 0
			}
		}
	}
}

func (mem *elfMemory) vcsLibInit() {
	switch mem.strongarm.state {
	case 0:
		mem.gpio.B[fromArm_Opcode] = 0x00
		mem.strongarm.state++
	case 3:
		mem.gpio.B[fromArm_Opcode] = 0x10
		mem.setNextRomAddress(0x1000)
		mem.endStrongArmFunction()
	default:
		mem.strongarm.state++
	}
}
