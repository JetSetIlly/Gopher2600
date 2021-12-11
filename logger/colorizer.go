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

package logger

import (
	"io"
	"strings"

	"github.com/jetsetilly/gopher2600/debugger/terminal/colorterm/easyterm/ansi"
)

// Colorizer applies basic coloring rules to logging output.
type Colorizer struct {
	out io.Writer
}

// NewColorizer is the preferred method if initialisation for the Colorizer type.
func NewColorizer(out io.Writer) Colorizer {
	return Colorizer{out: out}
}

// Write implements the io.Writer interface.
func (c Colorizer) Write(p []byte) (n int, err error) {
	n = 0

	l := strings.Split(strings.TrimSpace(string(p)), "\n")
	if len(l) == 0 {
		return n, nil
	}

	m, err := c.out.Write([]byte(l[0] + "\n"))
	if err != nil {
		return n + m, err
	}
	n += m

	if len(l) == 1 {
		return n, nil
	}

	m, err = c.out.Write([]byte(ansi.DimPens["red"]))
	if err != nil {
		return n + m, err
	}

	for _, s := range l[1:] {
		m, err := c.out.Write([]byte(s + "\n"))
		if err != nil {
			return n + m, err
		}
		n += m
	}

	defer func() {
		_, _ = c.out.Write([]byte(ansi.NormalPen))
	}()

	return n, nil
}
