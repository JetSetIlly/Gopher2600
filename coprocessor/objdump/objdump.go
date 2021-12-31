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

// package objdump file is a very basic parser for obj files as produced by
// "objdump -S" on the base elf file that is used to create a cartridge binary
//
// FindDataAccess() and FindProgramAccess() will return best-guess labels for
// the supplied address.
package objdump

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
)

type Entry struct {
	Asm string
	Src *string
}

// ObjDump contains the parsed information from the obj file.
type ObjDump struct {
	asm map[uint32]Entry
	src []string
}

const objFile = "armcode.obj"
const objFile_older = "custom2.obj"

func findObjDump(pathToROM string) *os.File {
	// current working directory
	sf, err := os.Open(objFile)
	if err == nil {
		return sf
	}

	dir := filepath.Dir(pathToROM)

	// same directory as binary
	sf, err = os.Open(filepath.Join(dir, objFile))
	if err == nil {
		return sf
	}

	// main sub-directory
	sf, err = os.Open(filepath.Join(dir, "main", objFile))
	if err == nil {
		return sf
	}

	// main/bin sub-directory
	sf, err = os.Open(filepath.Join(dir, "main", "bin", objFile))
	if err == nil {
		return sf
	}

	// custom/bin sub-directory. some older DPC+ sources uses this layout
	sf, err = os.Open(filepath.Join(dir, "custom", "bin", objFile_older))
	if err == nil {
		return sf
	}

	return nil
}

// NewObjDump loads and parses a obj file. Returns a new instance of Mapfile or
// any errors.
func NewObjDump(pathToROM string) (*ObjDump, error) {
	obj := &ObjDump{
		src: make([]string, 0),
		asm: make(map[uint32]Entry),
	}

	sf := findObjDump(pathToROM)
	if sf == nil {
		return nil, curated.Errorf("objfile: gcc .obj file not available (%s)", objFile)
	}
	defer sf.Close()

	data, err := io.ReadAll(sf)
	if err != nil {
		return nil, curated.Errorf("objfile: processing error: %v", err)
	}
	lines := strings.Split(string(data), "\n")

	var src strings.Builder

	asmMatch, err := regexp.Compile("^[[:xdigit:]]{8}:.*$")
	if err != nil {
		panic(fmt.Sprintf("objdump: %s", err.Error()))
	}

	srcIdx := -1

	for _, l := range lines {
		if asmMatch.Match([]byte(l)) {
			var addr uint32
			var asm string

			a, err := strconv.ParseUint(l[:8], 16, 32)
			if err == nil {
				if src.Len() > 0 {
					obj.src = append(obj.src, src.String())
					srcIdx++
					src.Reset()
				}

				addr = uint32(a)
				asm = strings.TrimSpace(l[9:])
				obj.asm[addr] = Entry{
					Asm: asm,
					Src: &obj.src[srcIdx],
				}
			}
		} else {
			var a, b string
			n, _ := fmt.Sscanf(l, "%8x <%s>\n", &a, &b)
			if n != 2 {
				src.WriteString(l)
				src.WriteRune('\n')
			}
		}
	}

	return obj, nil
}

// FindProgramAccess returns the program (function) label for the supplied
// address. Addresses may be in a range.
func (obj *ObjDump) FindProgramAccess(address uint32) string {
	return *obj.asm[address].Src
}
