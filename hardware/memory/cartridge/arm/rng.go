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

package arm

import (
	"math/rand"

	"github.com/jetsetilly/gopher2600/logger"
)

// the operation of the RNG unit in STM32 ARM packages can be found in the
// STM32 reference manual:
//
// https://www.st.com/resource/en/reference_manual/dm00031020-stm32f405-415-stm32f407-417-stm32f427-437-and-stm32f429-439-advanced-arm-based-32-bit-mcus-stmicroelectronics.pdf

// this RNG is just a sketch of the real RNG unit but for our purposes it's
// probably okay. this implementation basically returns a random 32bit number
// whenever the data register is read
type rng struct {
	// control register value
	control uint32

	// the status and data registers are handled differently in this
	// implementation. they are not writeable and will return a fixed value of
	// 0b1 in the case of the status register and a random number in the case
	// of the data register

	// extracted control register flags
	enabled          bool
	interruptEnabled bool
}

func (r *rng) reset() {
	r.control = 0x0
}

func (r *rng) write(addr uint32, val uint32) bool {
	switch addr {
	case 0x50060800:
		// control register
		r.control = val
		r.enabled = r.control&0b0100 == 0b0100
		r.interruptEnabled = r.control&0b1000 == 0b1000
	case 0x50060804:
		// status register
		logger.Logf("ARM7", "ignoring write to RNG status register (value of %08x)", val)
	case 0x50060808:
		// data register
		logger.Logf("ARM7", "ignoring write to RNG data register (value of %08x)", val)
	default:
		return false
	}

	return true
}

func (r *rng) read(addr uint32) (uint32, bool) {
	var val uint32

	switch addr {
	case 0x50060800:
		// control register
		val = r.control
	case 0x50060804:
		// status register. the low bit indicates that a random number is
		// ready. we're always ready to return a random number so we always
		// return 0b1
		val = 0b1
	case 0x50060808:
		// data register
		val = rand.Uint32()
	default:
		return 0, false
	}

	return val, true
}
