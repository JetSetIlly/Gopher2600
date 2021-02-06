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

package linter

import (
	"io"
	"strings"

	"github.com/jetsetilly/gopher2600/disassembly"
)

// Write performs a lint and writes the results output to io.Writer.
func Write(dsm *disassembly.Disassembly, output io.Writer) error {
	return dsm.IterateBlessed(output, func(e *disassembly.Entry) string {
		s := strings.Builder{}
		for _, r := range rules(e) {
			s.WriteString(r.String())
			s.WriteString("\n")
		}
		return s.String()
	})
}
