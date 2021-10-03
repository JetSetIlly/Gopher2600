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
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/resources"
)

const patchPath = "patches"

const neoComment = '-'
const neoSeparator = ":"

// CartridgeMemory applies the contents of a patch file to cartridge memory.
// Currently, patch file must be in the patches sub-directory of the
// resource path (see paths package).
func CartridgeMemory(mem *cartridge.Cartridge, patchFile string) (bool, error) {
	var err error

	p, err := resources.JoinPath(patchPath, patchFile)
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

	// read file
	buffer, err := io.ReadAll(f)
	if err != nil {
		return false, curated.Errorf("patch: %v", err)
	}

	if len(buffer) <= 1 {
		return false, nil
	}

	// if first character is a hyphen then we'll assume this is a "neo" style
	// patch file
	if buffer[0] == neoComment {
		err = neoStyle(mem, buffer)
		if err != nil {
			return false, curated.Errorf("patch: %v", err)
		}
		return true, nil
	}

	// otherwise assume it is a "cmp" style patch file
	err = cmpStyle(mem, buffer)
	if err != nil {
		return false, curated.Errorf("patch: %v", err)
	}
	return true, nil
}

// cmp -l <old_file> <new_file>.
func cmpStyle(mem *cartridge.Cartridge, buffer []byte) error {
	// walk through lines
	lines := strings.Split(string(buffer), "\n")
	for i, s := range lines {
		// ignore empty lines. cmp shouldn't output empty lines but the just in
		// case. pluse the last line will probably be empty
		if len(s) == 0 {
			continue
		}

		// split line into fields
		p := strings.Fields(s)

		// if there are not three fields then the file is malformed
		if len(p) != 3 {
			return curated.Errorf("cmp: line [%d]: malformed", i)
		}

		// ofset is stored as decimal
		offset, err := strconv.ParseUint(p[0], 10, 16)
		if err != nil {
			return curated.Errorf("cmp: line [%d]: %v", i, err)
		}

		// cmp counts from 1 but we count everything from zero
		offset--

		// old and patch bytes are stored as octal(!)
		old, err := strconv.ParseUint(p[1], 8, 8)
		if err != nil {
			return curated.Errorf("cmp: line [%d]: %v", i, err)
		}
		patch, err := strconv.ParseUint(p[2], 8, 8)
		if err != nil {
			return curated.Errorf("cmp: line [%d]: %v", i, err)
		}

		// check that the patch is correct
		o, _ := mem.Peek(uint16(offset))
		if o != uint8(old) {
			return curated.Errorf("cmp: line %d: byte at offset %04x does not match expected byte (%02x instead of %02x)", i, offset, o, old)
		}

		// patch memory
		err = mem.Patch(int(offset), uint8(patch))
		if err != nil {
			return curated.Errorf("cmp: %v", err)
		}
	}
	return nil
}

func neoStyle(mem *cartridge.Cartridge, buffer []byte) error {
	// walk through lines
	lines := strings.Split(string(buffer), "\n")
	for i, s := range lines {
		// ignore empty lines
		if len(s) == 0 {
			continue // for loop
		}

		// ignoring comment lines and lines starting with whitespace
		if s[0] == neoComment || unicode.IsSpace(rune(s[0])) {
			continue // for loop
		}

		pokeLine := strings.Split(s, neoSeparator)

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
				return curated.Errorf("neo: line %d: %v", i, err)
			}

			// advance offset
			offset++
		}
	}

	return nil
}
