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

	// number of cycles when flash/sram was being addressed
	FlashAccess float32
	SRAMAccess  float32

	// MAM is enabled *this* cycle
	MAMenabled bool

	// whether PC is in SRAM *this* cycle
	PCinSRAM bool
}

// multiline string.
func (c Cycles) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("I: %.0f\n", c.I))
	s.WriteString(fmt.Sprintf("C: %.0f\n", c.C))
	s.WriteString(fmt.Sprintf("N: %.0f\n", c.Npc+c.Ndata))
	s.WriteString(fmt.Sprintf("S: %.0f\n", c.Spc+c.Ndata))
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
	c.FlashAccess += n.FlashAccess
	c.SRAMAccess += n.SRAMAccess
	c.MAMenabled = n.MAMenabled
	c.PCinSRAM = n.PCinSRAM
}

func (c *Cycles) reset() {
	c.I = 0
	c.C = 0
	c.Npc = 0
	c.Spc = 0
	c.Ndata = 0
	c.Sdata = 0
	c.FlashAccess = 0
	c.SRAMAccess = 0
	c.MAMenabled = false
	c.PCinSRAM = false
}
