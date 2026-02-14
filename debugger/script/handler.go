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

package script

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Handler struct {
	lines []string
}

func (scr *Handler) Push(s string) {
	scr.lines = append(scr.lines, s)
}

func (scr *Handler) More() bool {
	return len(scr.lines) > 0
}

func (scr *Handler) Next() (string, bool) {
	if len(scr.lines) == 0 {
		return "", false
	}
	s := scr.lines[0]
	scr.lines = scr.lines[1:]
	return s, true
}

func (scr *Handler) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("script: no such file: %s", filename)
		}
		return fmt.Errorf("script: %w", err)
	}
	defer f.Close()

	s, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("script: %w", err)
	}

	// insert script before any other lines in the script
	lns := strings.Split(string(s), "\n")
	scr.lines = append(lns, scr.lines...)

	return nil
}
