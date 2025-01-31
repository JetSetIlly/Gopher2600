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
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/resources"
)

const patchPath = "patches"

const (
	neoComment   = '-'
	neoSeparator = ":"
)

var PatchFileNotFound = fmt.Errorf("patch: file not found")

// CartridgeMemoryFromHash applies the contents of a patch file in the
// .gopher2600/patches directory with the name the same as the cartridge hash
func CartridgeMemoryFromHash(cart *cartridge.Cartridge) error {
	p, err := resources.JoinPath(patchPath, cart.Hash)
	if err != nil {
		return fmt.Errorf("patch: %w", err)
	}
	return CartridgeMemoryFromFile(cart, p)
}

// CartridgeMemoryFromFile applies the contents of a patch file to cartridge memory.
func CartridgeMemoryFromFile(cart *cartridge.Cartridge, patchFile string) error {
	var err error

	// try to open patch file verbatim at first. if it's not open then try to
	f, err := os.Open(patchFile)
	if err != nil {
		var pathError *os.PathError
		if errors.As(err, &pathError) {
			return fmt.Errorf("%w: %s", PatchFileNotFound, patchFile)
		}
		return fmt.Errorf("patch: %w", err)
	}
	defer f.Close()

	// make sure we're at the beginning of the file
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("patch: %w", err)
	}

	// read file
	buffer, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("patch: %w", err)
	}

	if len(buffer) <= 1 {
		return fmt.Errorf("patch: file is empty")
	}

	lines := strings.Split(string(buffer), "\n")

	// find first non-empty line
	var l string
	for i := range lines {
		if len(lines[i]) > 0 {
			l = strings.TrimSpace(lines[i])
			break // for loop
		}
	}

	// if first character is a hyphen then we'll assume this is a "neo" style patch file
	if len(l) > 0 && l[0] == neoComment {
		err = neoStyle(cart, lines)
		if err != nil {
			return fmt.Errorf("patch: %w", err)
		}
		return nil
	}

	// otherwise assume it is a "cmp" style patch file
	err = cmpStyle(cart, lines)
	if err != nil {
		return fmt.Errorf("patch: %w", err)
	}
	return nil
}

// cmp -l <old_file> <new_file>.
func cmpStyle(cart *cartridge.Cartridge, lines []string) error {
	// walk through lines
	for i, s := range lines {
		// trim space of every line
		s = strings.TrimSpace(s)

		// ignore empty lines. cmp shouldn't output empty lines but the just in
		// case. pluse the last line will probably be empty
		if len(s) == 0 {
			continue
		}

		// split line into fields
		p := strings.Fields(s)

		// if there are not three fields then the file is malformed
		if len(p) != 3 {
			return fmt.Errorf("cmp: line %d: malformed", i)
		}

		// ofset is stored as decimal
		offset, err := strconv.ParseUint(p[0], 10, 16)
		if err != nil {
			return fmt.Errorf("cmp: line %d: %w", i, err)
		}

		// cmp counts from 1 but we count everything from zero
		offset--

		// old and patch bytes are stored as octal(!)
		old, err := strconv.ParseUint(p[1], 8, 8)
		if err != nil {
			return fmt.Errorf("cmp: line %d: %w", i, err)
		}
		patch, err := strconv.ParseUint(p[2], 8, 8)
		if err != nil {
			return fmt.Errorf("cmp: line %d: %w", i, err)
		}

		// check that the patch is correct
		o, _ := cart.Peek(uint16(offset))
		if o != uint8(old) {
			return fmt.Errorf("cmp: line %d: byte at offset %04x does not match expected byte (%02x instead of %02x)", i, offset, o, old)
		}

		// patch memory
		err = cart.Patch(int(offset), uint8(patch))
		if err != nil {
			return fmt.Errorf("cmp: %w", err)
		}
	}
	return nil
}

func neoStyle(cart *cartridge.Cartridge, lines []string) error {
	// walk through lines
	for i, s := range lines {
		// trim space of every line
		s = strings.TrimSpace(s)

		// ignore empty lines
		if len(s) == 0 {
			continue // for loop
		}

		// ignoring comment lines and lines starting with whitespace
		if s[0] == neoComment || unicode.IsSpace(rune(s[0])) {
			continue // for loop
		}

		p := strings.Split(s, neoSeparator)

		// ignore any lines that don't match the required [offset: values...] format
		if len(p) != 2 {
			continue // for loop
		}

		// trim space around each poke line part
		p[0] = strings.TrimSpace(p[0])
		p[1] = strings.TrimSpace(p[1])

		// parse offset
		offset, err := strconv.ParseInt(p[0], 16, 16)
		if err != nil {
			continue // for loop
		}

		// split values into parts
		values := strings.Split(p[1], " ")
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
			err = cart.Patch(int(offset), uint8(v))
			if err != nil {
				return fmt.Errorf("neo: line %d: %w", i, err)
			}

			// advance offset
			offset++
		}
	}

	return nil
}
