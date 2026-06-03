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

package disassembly

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jetsetilly/gopher2600/test"
)

func TestQuickDecode(t *testing.T) {
	dec := newQuickDecode()

	ram := make([]uint8, 128)
	ram[0] = 0xea // nop

	res, err := dec.decode(0x80, ram, 0x80)
	test.ExpectSuccess(t, err)
	test.ExpectSuccess(t, res.Final)
	test.ExpectEquality(t, fmt.Sprintf("%#v", res.Defn), "&instructions.Definition{OpCode:0xea, Operator:0, Bytes:1, Cycles:2, AddressingMode:0, PageSensitive:false, Effect:0, Undocumented:false, Stability:0}")

	dsm := formatResultQuick(res, EntryLevelBlessed)
	test.ExpectEquality(t, strings.TrimSpace(dsm.String()), "$0080 nop")
}
