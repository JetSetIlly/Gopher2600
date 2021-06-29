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

package arm7tdmi

import (
	"math/bits"
)

// Cycles records the number of cycles for (a) an individual instruction and
// (b) a collection of instructions. It is the underlying type for both
// mapper.CartCoProcCycleDetails and mapper.CartCoProcExecutionSummary.
//
// In the context of the execution summary the boolean variables take the most
// recent value held by an individual instruction. The float32 value are a
// summation of all instructions executed in a single call to ARM.Run().
type Cycles struct {
	// prot0 == 0
	Iopcode float32
	Nopcode float32
	Sopcode float32

	// prot0 == 1
	Idata float32
	Ndata float32
	Sdata float32

	// There are no coprocessor cycles in this emulation.

	// MAMCR value for *this* cycle. only values 0, 1 or 2 are valid. treat any
	// other value as 0
	MAMCR uint32

	// whether to wait for pc/data cycles
	waitForInstruction bool
	waitForData        bool

	// whether instruction has written
	writeData bool

	// data transfer size
	sizeData int // 0, 1, 2 or 4
}

// add two Cycles instances together. float32 fields are added together. other
// fields are not added.
func (c *Cycles) add(n Cycles) {
	c.Iopcode += n.Iopcode
	c.Nopcode += n.Nopcode
	c.Sopcode += n.Sopcode
	c.Idata += n.Idata
	c.Ndata += n.Ndata
	c.Sdata += n.Sdata
}

func (c *Cycles) reset() {
	c.Iopcode = 0
	c.Nopcode = 0
	c.Sopcode = 0
	c.Idata = 0
	c.Ndata = 0
	c.Sdata = 0
	c.MAMCR = 0
	c.waitForInstruction = false
	c.waitForData = false
	c.writeData = false
	c.sizeData = 0
}

// simple (unstretched) cycle count.
func (c *Cycles) count() float32 {
	return c.Iopcode + c.Idata + c.Nopcode + c.Ndata + c.Sopcode + c.Sdata
}

// shift should be false when shift amount if zero, even when the instruction
// is a shift - the zero amount means the shift does not happen.
func (c *Cycles) dataOperations(pc bool, shift bool) {
	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"

	// I cycle merged:
	//
	// "When a register specifies the shift length, an additional data path cycle occurs before
	// the data operation to copy the bottom 8 bits of that register into a holding latch in the
	// barrel shifter. The instruction prefetch occurs during this first cycle. The operation cycle
	// is internal (it does not request memory). Because the address remains stable through
	// both cycles, the memory manager can merge this internal cycle with the following
	// sequential access."

	if shift {
		if pc {
			// c.Iopcode++ // merged
			c.Ndata++
			c.Sopcode++
			c.Sopcode++
		} else {
			// c.Iopcode++ // merged
			c.Sdata++
		}
	} else {
		if pc {
			c.Nopcode++
			c.Sopcode++
			c.Sopcode++
		} else {
			c.Sopcode++
		}
	}
}

func (c *Cycles) mul(operand uint32) {
	// "7.7 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	//  and
	// "7.2 Instruction Cycle Count Summary"  in "ARM7TDMI-S Technical
	// Reference Manual r4p3" ...

	p := bits.OnesCount32(operand & 0xffffff00)
	if p == 0 || p == 24 {
		// ... Is 1 if bits [32:8] of the multiplier operand are all zero or one.
		c.Iopcode++
	} else {
		p := bits.OnesCount32(operand & 0xffff0000)
		if p == 0 || p == 16 {
			// ... Is 2 if bits [32:16] of the multiplier operand are all zero or one.
			c.Iopcode++
			c.Idata++
		} else {
			p := bits.OnesCount32(operand & 0xff000000)
			if p == 0 || p == 8 {
				// ... Is 3 if bits [31:24] of the multiplier operand are all zero or one.
				c.Iopcode++
				c.Idata += 2
			} else {
				// ... Is 4 otherwise.
				c.Iopcode++
				c.Idata += 3
			}
		}
	}

	c.Sdata++
}

func (c *Cycles) loadRegister(pc bool) {
	// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"

	// I cycle merged:
	//
	// "During the third cycle, the ARM7TDMI-S processor transfers the data to the
	// destination register. (External memory is not used.) Normally, the ARM7TDMI-S
	// core merges this third cycle with the next prefetch to form one memory N-cycle."

	if pc {
		c.Nopcode++
		// c.Idata++ // merged
		c.Ndata++
		c.Sopcode++
		c.Sopcode++
	} else {
		c.Nopcode++
		// c.Idata++ // merged
		c.Sdata++
	}
}

func (c *Cycles) storeRegister() {
	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"

	c.Nopcode++
	// c.Ndata++ // merged N cycles
}

// num is the total number of registers being loaded. if pc is true then that
// says the one of those registers is the PC. in other words if pc is true then
// num must be at least one.
func (c *Cycles) loadMultipleRegisters(pc bool, num int) {
	// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// and
	// "4.11" in "ARM7TDMI Data Sheet"

	// I cycle merged:
	//
	// "During the fourth and final (internal) cycle, the ARM7TDMI-S core moves the
	// last word to its destination register. The last cycle can be merged with the next
	// instruction prefetch to form a single memory N-cycle."

	if pc {
		c.Nopcode++
		for i := 0; i < num-1; i++ {
			c.Sdata++
		}
		// c.Idata++ // merged
		c.Ndata++
		c.Sopcode++
		c.Sopcode++
	} else {
		c.Nopcode++
		for i := 0; i < num-1; i++ { // last S cycle merged
			c.Sdata++
		}
		c.Idata++
		// c.Sdata++ // merged
	}
}

func (c *Cycles) storeMultipleRegisters(num int) {
	// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"

	c.Nopcode++
	for i := 0; i < num-1; i++ {
		c.Sdata++
	}
	if num > 1 {
		c.Ndata++ // merged N cycles
	}
}

func (c *Cycles) conditionalBranch(branch bool) {
	// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"

	if branch {
		c.Nopcode++
		c.Sopcode++
		c.Sopcode++
	} else {
		c.Sopcode++
	}
}

func (c *Cycles) unconditionalBranch() {
	// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"

	c.Nopcode++
	c.Sopcode++
	c.Sopcode++
}

func (c *Cycles) thumbBranch() {
	// "7.4 Thumb Branch With Link" in "ARM7TDMI-S Technical Reference Manual r4p3"

	// S cycle for first instruction.
	c.Sopcode++

	// second instruction
	c.Nopcode++
	c.Sopcode++
	c.Sopcode++
}
