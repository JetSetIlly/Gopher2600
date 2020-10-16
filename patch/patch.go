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

package patch

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/paths"
)

const patchPath = "patches"

const commentLeader = '-'
const pokeLineSeparator = ":"

// CartridgeMemory applies the contents of a patch file to cartridge memory.
// Currently, patch file must be in the patches sub-directory of the
// resource path (see paths package).
func CartridgeMemory(mem *cartridge.Cartridge, patchFile string) (bool, error) {
	var err error

	p, err := paths.ResourcePath(patchPath, patchFile)
	if err != nil {
		return false, curated.Errorf("patch: %v", err)
	}

	f, err := os.Open(p)
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			return false, curated.Errorf("patch: %v", fmt.Sprintf("patch file not found (%s)", p))
		}
		return false, curated.Errorf("patch: %v", err)
	}
	defer f.Close()

	// make sure we're at the beginning of the file
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return false, curated.Errorf("patch: %v", err)
	}

	buffer, err := ioutil.ReadAll(f)
	if err != nil {
		return false, curated.Errorf("patch: %v", err)
	}

	// once a patch has been made then we'll flip patched to true and return it
	// to the calling function
	patched := false

	// walk through lines
	lines := strings.Split(string(buffer), "\n")
	for i := 0; i < len(lines); i++ {
		// ignore empty lines
		if len(lines[i]) == 0 {
			continue // for loop
		}

		// ignoring comment lines and lines starting with whitespace
		if lines[i][0] == commentLeader || unicode.IsSpace(rune(lines[i][0])) {
			continue // for loop
		}

		pokeLine := strings.Split(lines[i], pokeLineSeparator)

		// ignore any lines that don't match the required [offset: values...] format
		if len(pokeLine) != 2 {
			continue // for loop
		}

		// trim space around each poke line part
		pokeLine[0] = strings.TrimSpace(pokeLine[0])
		pokeLine[1] = strings.TrimSpace(pokeLine[1])

		// parse offset
		offset, err := strconv.ParseInt(pokeLine[0], 16, 16)
		if err != nil {
			continue // for loop
		}

		// split values into parts
		values := strings.Split(pokeLine[1], " ")
		for j := 0; j < len(values); j++ {
			// trim space around each value
			values[j] = strings.TrimSpace(values[j])

			// ignore empty fields
			if values[j] == "" {
				continue // inner for loop
			}

			// covert data
			v, err := strconv.ParseUint(values[j], 16, 8)
			if err != nil {
				continue // inner for loop
			}

			// patch memory
			err = mem.Patch(int(offset), uint8(v))
			if err != nil {
				return patched, curated.Errorf("patch: %v", err)
			}
			patched = true

			// advance offset
			offset++
		}
	}

	return patched, nil
}
