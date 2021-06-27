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
	"fmt"
	"strings"
)

// Cycles records the number of cycles for (a) an individual instruction and
// (b) a collection of instructions. It is the underlying type for both
// mapper.CartCoProcCycleDetails and mapper.CartCoProcExecutionSummary.
//
// In the context of the execution summary the boolean variables take the most
// recent value held by an individual instruction. The float32 value are a
// summation of all instructions executed in a single call to ARM.Run().
type Cycles struct {
	I     float32
	C     float32
	Npc   float32
	Spc   float32
	Ndata float32
	Sdata float32

	// merged cycles are cycles that happen but can be done in parallel with
	// another cycle, meaning that they can be discounted from cycle counts.
	//
	// Imerged cycles can be seen in formats that use the cycle profile
	// described in sections 7.6 and 7.10 of the ARM7TDMI technical reference
	// manual.
	//
	// Spcmerged cycles can be seen in formats that use the cycle profile
	// described in sections 7.8 ARM7TDMI technical reference manual.
	Imerged   float32
	Spcmerged float32

	// MAMCR value for *this* cycle. only values 0, 1 or 2 are valid. treat any
	// other value as 0
	MAMCR uint32

	// whether PC is in SRAM
	PCinSRAM bool

	// whether any data rad is from SRAM
	DataInSRAM bool
}

// multiline string.
func (c Cycles) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("I: %.0f\n", c.I+c.Imerged))
	s.WriteString(fmt.Sprintf("C: %.0f\n", c.C))
	s.WriteString(fmt.Sprintf("N: %.0f\n", c.Npc+c.Ndata))
	s.WriteString(fmt.Sprintf("S: %.0f\n", c.Spc+c.Sdata+c.Spcmerged))
	return s.String()
}

// add one Cycles instance to another. Boolean fields take the value of the
// instance being added. Float32 fields are summed.
func (c *Cycles) add(n Cycles) {
	c.I += n.I
	c.C += n.C
	c.Npc += n.Npc
	c.Spc += n.Spc
	c.Ndata += n.Ndata
	c.Sdata += n.Sdata
	c.Imerged += n.Imerged
	c.Spcmerged += n.Spcmerged
	c.MAMCR = n.MAMCR
	c.PCinSRAM = n.PCinSRAM
	c.DataInSRAM = n.DataInSRAM
}

func (c *Cycles) reset() {
	c.I = 0
	c.C = 0
	c.Npc = 0
	c.Spc = 0
	c.Ndata = 0
	c.Sdata = 0
	c.Imerged = 0
	c.Spcmerged = 0
	c.MAMCR = 0
	c.PCinSRAM = false
	c.DataInSRAM = false
}

// simple (unstretched) cycle count.
func (c *Cycles) count() float32 {
	return c.I + c.Imerged + c.C + c.Npc + c.Ndata + c.Spc + c.Sdata + c.Spcmerged
}
