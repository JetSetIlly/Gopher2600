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

type disasmLevel int

const (
	disasmFull disasmLevel = iota
	disasmNotes
	disasmNone
)

type cycles struct {
	I float32
	C float32
	N float32
	S float32
}

func (c *cycles) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("I: %.0f\n", c.I))
	s.WriteString(fmt.Sprintf("C: %.0f\n", c.C))
	s.WriteString(fmt.Sprintf("N: %.0f\n", c.N))
	s.WriteString(fmt.Sprintf("S: %.0f\n", c.S))
	return s.String()
}

func (c *cycles) sum() float32 {
	return c.I + c.C + (2 * c.N) + c.S
}

func (c *cycles) reset() {
	c.I = 0
	c.C = 0
	c.N = 0
	c.S = 0
}
