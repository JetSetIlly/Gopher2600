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
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// void vcsCopyOverblankToRiotRam()
func vcsCopyOverblankToRiotRam(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		mem.strongarm.running.state++
		mem.strongarm.running.counter = 0
		mem.strongarm.running.subCounter = 0
		fallthrough
	case 1:
		if mem.strongarm.running.counter >= len(overblank) {
			mem.endStrongArmFunction()
		} else {
			switch mem.strongarm.running.subCounter {
			case 0:
				if mem.injectRomByte(0xa9) {
					mem.strongarm.running.subCounter++
				}
			case 1:
				if mem.injectRomByte(overblank[mem.strongarm.running.counter]) {
					mem.strongarm.running.subCounter++
				}
			case 2:
				if mem.injectRomByte(0x85) {
					mem.strongarm.running.subCounter++
				}
			case 3:
				if mem.injectRomByte(uint8(0x80 + mem.strongarm.running.counter)) {
					mem.strongarm.running.subCounter++
				}
			case 4:
				if mem.yieldDataBus(uint16(0x80 + mem.strongarm.running.counter)) {
					mem.strongarm.running.counter++
					mem.strongarm.running.subCounter = 0
				}
			}
		}
	}
}

// sequence for initialisation triggered by the accessing of the reset address.
// the sequence is very strict so there is no need for coordination with
// setNextAddress() or injectRomByte()
func vcsLibInit(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		mem.gpio.data[DATA_ODR] = 0x00
		mem.strongarm.running.state++
	case 1:
		mem.gpio.data[DATA_ODR] = 0x10
		mem.setNextRomAddress(0x1000)
		mem.endStrongArmFunction()
	}
}

func vcsInitBusStuffing(mem *elfMemory) {
	mem.usesBusStuffing = true
}

func (str *strongArmState) updateLookupTables() {
	for i := 0; i < 256; i++ {
		if uint8(i)&str.correctionMaskHi == str.correctionMaskHi {
			if uint8(i)&str.correctionMaskLo == str.correctionMaskLo {
				str.opcodeLookup[i] = 0x84
			} else {
				str.opcodeLookup[i] = 0x86
			}
		} else {
			if uint8(i)&str.correctionMaskLo == str.correctionMaskLo {
				str.opcodeLookup[i] = 0x85
			} else {
				str.opcodeLookup[i] = 0x87
			}
		}

		mode := uint8(i) ^ str.lowMask

		// never drive the bits that get corrected by opcodes above
		mode &= ^str.correctionMaskLo
		mode &= ^str.correctionMaskHi

		str.modeLookup[i] = ((mode & 0x80) << 7) |
			((mode & 0x40) << 6) |
			((mode & 0x20) << 5) |
			((mode & 0x10) << 4) |
			((mode & 0x08) << 3) |
			((mode & 0x04) << 2) |
			((mode & 0x02) << 1) |
			(mode & 0x01)
	}
}

// initialise state ready for bus stuffing. we know bus stuffing is used if the
// vcsWrite3() function has been detected (during relocation).
func (mem *elfMemory) vcsInitBusStuffing() {
	if !mem.usesBusStuffing {
		logger.Log(mem.env, "ELF", "ROM does not use any bus stuffing instructions")
		return
	}

	logger.Log(mem.env, "ELF", "ROM uses bus stuffing instructions")

	mem.strongarm.lowMask = 0xff
	mem.strongarm.correctionMaskHi = 0x00
	mem.strongarm.correctionMaskLo = 0x00
	mem.strongarm.updateLookupTables()
}

// void vcsWrite3(uint8_t ZP, uint8_t data)
func vcsWrite3(mem *elfMemory) {
	zp := uint8(mem.strongarm.running.registers[0])
	data := uint8(mem.strongarm.running.registers[1])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(mem.strongarm.opcodeLookup[data]) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(zp) {
			mem.strongarm.running.state++
		}
		mem.injectBusStuff(data)
	case 2:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsJmp3()
func vcsJmp3(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x4c) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(0x00) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(0x10) {
			mem.endStrongArmFunction()
			mem.setNextRomAddress(0x1000)
		}
	}
}

// void vcsLda2(uint8_t data)
func vcsLda2(mem *elfMemory) {
	data := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0xa9) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(data) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsSta3(uint8_t ZP)
func vcsSta3(mem *elfMemory) {
	zp := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x85) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(zp) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// uint8_t snoopDataBus(uint16_t address)
func snoopDataBus(mem *elfMemory) {
	addrIn := uint16(mem.gpio.data[ADDR_IDR])
	addrIn |= uint16(mem.gpio.data[ADDR_IDR+1]) << 8
	addrIn &= memorymap.Memtop

	if addrIn == mem.strongarm.nextRomAddress {
		// setting return value
		mem.arm.RegisterSet(0, uint32(mem.gpio.data[DATA_IDR]))
		mem.endStrongArmFunction()
	}

	// note that this implementation of snoopDataBus is missing the "give
	// peripheral time to respond" loop that we see in the real vcsLib
}

// snoopDataBus is significantly different when streaming is enabled
func snoopDataBus_streaming(mem *elfMemory, addr uint16) {
	if addr == mem.strongarm.nextRomAddress {
		mem.arm.RegisterSet(0, uint32(mem.gpio.data[DATA_IDR]))
		mem.stream.snoopDataBus = false
	}
}

// uint8_t vcsRead4(uint16_t address)
func vcsRead4(mem *elfMemory) {
	address := uint16(mem.strongarm.running.registers[0])
	address &= memorymap.Memtop

	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0xad) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(uint8(address)) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(uint8(address >> 8)) {
			if mem.stream.active {
				mem.endStrongArmFunction()
				mem.stream.startDrain()
				mem.stream.snoopDataBus = true
			} else {
				mem.setStrongArmFunction(snoopDataBus)
			}
		}
	}
}

// void vcsStartOverblank()
func vcsStartOverblank(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x4c) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(0x80) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(0x00) {
			mem.strongarm.running.state++
		}
	case 3:
		if mem.yieldDataBus(uint16(0x0080)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsEndOverblank()
func vcsEndOverblank(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		mem.setNextRomAddress(0x1fff)
		if mem.injectRomByte(0x00) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.yieldDataBus(uint16(0x00ac)) {
			mem.setNextRomAddress(0x1000)
			mem.endStrongArmFunction()
		}
	}
}

// void vcsLdaForBusStuff2()
func vcsLdaForBusStuff2(mem *elfMemory) {
	mem.setStrongArmFunction(vcsLda2, 0xff)
	mem.strongarm.running.function(mem)
}

// void vcsLdxForBusStuff2()
func vcsLdxForBusStuff2(mem *elfMemory) {
	mem.setStrongArmFunction(vcsLdx2, 0xff)
	mem.strongarm.running.function(mem)
}

// void vcsLdyForBusStuff2()
func vcsLdyForBusStuff2(mem *elfMemory) {
	mem.setStrongArmFunction(vcsLdy2, 0xff)
	mem.strongarm.running.function(mem)
}

// void vcsWrite5(uint8_t ZP, uint8_t data)
func vcsWrite5(mem *elfMemory) {
	zp := uint8(mem.strongarm.running.registers[0])
	data := uint8(mem.strongarm.running.registers[1])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0xa9) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(data) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(0x85) {
			mem.strongarm.running.state++
		}
	case 3:
		if mem.injectRomByte(zp) {
			mem.strongarm.running.state++
		}
	case 4:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsWrite6(uint8_t address, uint8_t data)
func vcsWrite6(mem *elfMemory) {
	address := uint16(mem.strongarm.running.registers[0])
	data := uint8(mem.strongarm.running.registers[1])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0xa9) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(data) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(0x8d) {
			mem.strongarm.running.state++
		}
	case 3:
		if mem.injectRomByte(uint8(address & 0xff)) {
			mem.strongarm.running.state++
		}
	case 4:
		if mem.injectRomByte(uint8(address >> 8)) {
			mem.strongarm.running.state++
		}
	case 5:
		if mem.yieldDataBus(address) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsLdx2(uint8_t data)
func vcsLdx2(mem *elfMemory) {
	data := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0xa2) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(data) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsLdy2(uint8_t data)
func vcsLdy2(mem *elfMemory) {
	data := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0xa0) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(data) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsSta4(uint8_t ZP)
func vcsSta4(mem *elfMemory) {
	zp := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x8d) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(zp) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(0) {
			mem.strongarm.running.state++
		}
	case 3:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsSax3(uint8_t ZP)
func vcsSax3(mem *elfMemory) {
	zp := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x87) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(zp) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsStx3(uint8_t ZP)
func vcsStx3(mem *elfMemory) {
	zp := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x86) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(zp) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsStx4(uint8_t ZP)
func vcsStx4(mem *elfMemory) {
	zp := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x8e) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(zp) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(0) {
			mem.strongarm.running.state++
		}
	case 3:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsSty3(uint8_t ZP)
func vcsSty3(mem *elfMemory) {
	zp := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x84) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(zp) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsSty4(uint8_t ZP)
func vcsSty4(mem *elfMemory) {
	zp := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x8c) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(zp) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(0) {
			mem.strongarm.running.state++
		}
	case 3:
		if mem.yieldDataBus(uint16(zp)) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsJsr6(uint16_t target)
func vcsJsr6(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x20) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(uint8(mem.strongarm.running.registers[0])) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(uint8(mem.strongarm.running.registers[0] >> 8)) {
			mem.gpio.data[ADDR_IDR] = uint8(mem.strongarm.running.registers[0])
			mem.gpio.data[ADDR_IDR+1] = uint8(mem.strongarm.running.registers[0] >> 8)
			mem.gpio.data[ADDR_IDR+2] = uint8(mem.strongarm.running.registers[0] >> 16)
			mem.gpio.data[ADDR_IDR+3] = uint8(mem.strongarm.running.registers[0] >> 24)

			mem.endStrongArmFunction()
		}
	}
}

// void vcsNop2()
func vcsNop2(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0xea) {
			mem.strongarm.running.state++
		}
	case 1:
		mem.endStrongArmFunction()
	}
}

// void vcsNop2n(uint16_t n)
func vcsNop2n(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0xea) {
			mem.strongarm.nextRomAddress += uint16(mem.strongarm.running.registers[0]) - 1
			mem.strongarm.running.state++
		}
	case 1:
		mem.endStrongArmFunction()
	}
}

// void vcsTxs2()
func vcsTxs2(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x9a) {
			mem.strongarm.running.state++
		}
	case 1:
		mem.endStrongArmFunction()
	}
}

// void vcsPha3()
func vcsPha3(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x48) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.yieldDataBusToStack() {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsPhp3()
func vcsPhp3(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x08) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.yieldDataBusToStack() {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsPlp4()
func vcsPlp4(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x28) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.yieldDataBusToStack() {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsPla4()
func vcsPla4(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x68) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.yieldDataBusToStack() {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsPlp4Ex(uint8_t data)
func vcsPlp4Ex(mem *elfMemory) {
	data := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x28) {
			mem.strongarm.running.state++
		}
		mem.injectBusStuff(data & 0x3f)
	case 1:
		if mem.yieldDataBusToStack() {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsPla4Ex(uint8_t data)
func vcsPla4Ex(mem *elfMemory) {
	data := uint8(mem.strongarm.running.registers[0])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x68) {
			mem.strongarm.running.state++
		}
		mem.injectBusStuff(data & 0x3f)
	case 1:
		if mem.yieldDataBusToStack() {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsWaitForAddress(uint16_t address)
func vcsWaitForAddress(mem *elfMemory) {
	addrIn := uint16(mem.gpio.data[ADDR_IDR])
	addrIn |= uint16(mem.gpio.data[ADDR_IDR+1]) << 8
	addrIn &= memorymap.Memtop
	address := uint16(mem.strongarm.running.registers[0])
	if addrIn == address {
		mem.endStrongArmFunction()
	}
}

// void vcsJmpToRam3(uint16_t address)
func vcsJmpToRam3(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(0x4c) {
			mem.strongarm.running.state++
		}
	case 1:
		address := uint16(mem.strongarm.running.registers[0])
		if mem.injectRomByte(uint8(address)) {
			mem.strongarm.running.state++
		}
	case 2:
		address := uint16(mem.strongarm.running.registers[0])
		if mem.injectRomByte(uint8(address >> 8)) {
			mem.strongarm.running.state++
		}
	case 3:
		address := uint16(mem.strongarm.running.registers[0])
		if mem.yieldDataBus(address) {
			mem.endStrongArmFunction()
			mem.setNextRomAddress(address)
			mem.arm.Interrupt()
			mem.stream.startDrain()
		}
	}
}

// void vcsWrite4(uint16_t address, uint8_t data)
func vcsWrite4(mem *elfMemory) {
	address := uint16(mem.strongarm.running.registers[0])
	data := uint8(mem.strongarm.running.registers[1])
	switch mem.strongarm.running.state {
	case 0:
		if mem.injectRomByte(mem.strongarm.opcodeLookup[data] + 8) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.injectRomByte(uint8(address)) {
			mem.strongarm.running.state++
		}
	case 2:
		if mem.injectRomByte(uint8(address >> 8)) {
			mem.strongarm.running.state++
		}
		mem.injectBusStuff(data)
	case 3:
		if mem.yieldDataBus(address) {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsPokeRomByte(uint16_t uint16_t address, uint8_t data)
func vcsPokeRomByte(mem *elfMemory) {
	switch mem.strongarm.running.state {
	case 0:
		address := uint16(mem.strongarm.running.registers[0])
		data := uint8(mem.strongarm.running.registers[1])
		mem.setNextRomAddress(address)
		if mem.injectRomByte(data) {
			mem.strongarm.running.state++
		}
	case 1:
		if mem.yieldDataBusToStack() {
			mem.endStrongArmFunction()
		}
	}
}

// void vcsSetNextAddress(uint16_t address)
func vcsSetNextAddress(mem *elfMemory) {
	address := uint16(mem.strongarm.running.registers[0])
	mem.setNextRomAddress(address)
}
