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
	"bytes"
	"fmt"
)

// String returns a very basic representation of an Entry. Provided for
// convenience. Probably not of any use except for the simplest of tools.
//
// See StringColumnated() for a fancier option.
func (e *Entry) String() string {
	operand := e.Operand.genString()
	return fmt.Sprintf("%s %s %s", e.Address, e.Operator, operand)
}

// ColumnAttr controls what is included in the string returned by Entry.StringColumnated().
type ColumnAttr struct {
	ByteCode bool
	Cycles   bool
	Label    bool
}

// StringColumnated returns a columnated string representation of the Entry.
//
// Trailing newline is not included. However, if attr.Label is true then a
// newline will be added after any label.
func (e *Entry) StringColumnated(attr ColumnAttr) string {
	b := &bytes.Buffer{}

	if e == nil {
		return ""
	}

	if e.Level < EntryLevelBlessed {
		return ""
	}

	if attr.Label {
		if e.Label.String() != "" {
			b.Write([]byte(e.GetField(FldLabel)))
			b.Write([]byte("\n"))
		}
	}

	if attr.ByteCode {
		b.Write([]byte(e.GetField(FldBytecode)))
		b.Write([]byte(" "))
	}

	b.Write([]byte(e.GetField(FldAddress)))
	b.Write([]byte(" "))
	b.Write([]byte(e.GetField(FldOperator)))
	b.Write([]byte(" "))
	b.Write([]byte(e.GetField(FldOperand)))

	if attr.Cycles {
		b.Write([]byte(" "))
		b.Write([]byte(e.GetField(FldCycles)))
	}

	return b.String()
}
