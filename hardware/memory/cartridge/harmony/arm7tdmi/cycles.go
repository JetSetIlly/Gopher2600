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

// Clock speeds inside the arm7 sub-system.
const (
	InternalClk     = 70.0 // Mhz
	FlashAccessTime = 28.6 // ns
	SRAMAccessTime  = 10.0 // ns

	// strict access ratios calculated by:
	//
	//    InternalClk / (1000 / access time)

	FlashAccessRatio = 2.45
	SRAMAccessRatio  = 1.0 // floor of 1.0
	MAMAccessRatio   = 1.0
)

type cycles struct {
	I     float32
	C     float32
	Npc   float32
	Spc   float32
	Ndata float32
	Sdata float32

	MAMenabled bool
	PCinSRAM   bool
}

func (c *cycles) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("I: %.0f\n", c.I))
	s.WriteString(fmt.Sprintf("C: %.0f\n", c.C))
	s.WriteString(fmt.Sprintf("N: %.0f\n", c.Npc+c.Ndata))
	s.WriteString(fmt.Sprintf("S: %.0f\n", c.Spc+c.Ndata))
	return s.String()
}

func (c *cycles) sum(pcaddr uint32, mam bool) float32 {
	c.MAMenabled = mam
	c.PCinSRAM = pcaddr > Flash64kMemtop

	t := c.I + c.C

	if mam {
		t += (c.Npc * MAMAccessRatio) + (c.Spc * MAMAccessRatio)
	} else if c.PCinSRAM {
		t += (c.Npc * SRAMAccessRatio) + (c.Spc * SRAMAccessRatio)
	} else {
		t += (c.Npc * FlashAccessRatio) + (c.Spc * FlashAccessRatio)
	}

	// assumption: all data acces is to SRAM
	t += (c.Ndata * SRAMAccessRatio) + (c.Sdata * SRAMAccessRatio)

	return t
}

func (c *cycles) reset() {
	c.I = 0
	c.C = 0
	c.Npc = 0
	c.Spc = 0
	c.Ndata = 0
	c.Sdata = 0

	// no need to reset flags
}
