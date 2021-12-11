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

// package map file is a very basic parser for obj files as produced by
// "objdump -S" on the base elf file that is used to create a cartridge binary
//
// FindDataAccess() and FindProgramAccess() will return best-guess labels for
// the supplied address.
package objdump

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
)

// ObjDump contains the parsed information from the map file.
type ObjDump struct {
	lines []string
}

const mapFile = "armcode.obj"

func findObjDump(pathToROM string) *os.File {
	// current working directory
	sf, err := os.Open(mapFile)
	if err == nil {
		return sf
	}

	dir := filepath.Dir(pathToROM)

	// same directory as binary
	sf, err = os.Open(filepath.Join(dir, mapFile))
	if err == nil {
		return sf
	}

	// main sub-directory
	sf, err = os.Open(filepath.Join(dir, "main", mapFile))
	if err == nil {
		return sf
	}

	// main/bin sub-directory
	sf, err = os.Open(filepath.Join(dir, "main", "bin", mapFile))
	if err == nil {
		return sf
	}

	return nil
}

// NewObjDump loads and parses a map file. Returns a new instance of Mapfile or
// any errors.
func NewObjDump(pathToROM string) (*ObjDump, error) {
	obj := &ObjDump{}

	sf := findObjDump(pathToROM)
	if sf == nil {
		return nil, curated.Errorf("mapfile: gcc .map file not available (%s)", mapFile)
	}
	defer sf.Close()

	data, err := io.ReadAll(sf)
	if err != nil {
		return nil, curated.Errorf("mapfile: processing error: %v", err)
	}
	obj.lines = strings.Split(string(data), "\n")

	return obj, nil
}

// FindDataAccess returns the data label for the supplied address. Addresses
// will be matched exactly.
func (obj *ObjDump) FindDataAccess(address uint32) string {
	return ""
}

// FindProgramAccess returns the program (function) label for the supplied
// address. Addresses may be in a range.
func (obj *ObjDump) FindProgramAccess(address uint32) string {
	var start int
	var end int

	src := false

	for i, l := range obj.lines {
		flds := strings.Fields(l)
		if len(flds) == 0 {
			continue
		}

		if len(flds) >= 2 {
			if flds[0][len(flds[0])-1] == ':' {
				if src {
					end = i
					src = false
				}

				v, err := strconv.ParseInt(flds[0][:len(flds[0])-1], 16, 64)
				pc := uint32(v)
				if err == nil {
					if pc == address {
						break // for loop
					}
				}
			} else if !src {
				start = i
				src = true
			}
		}

	}

	s := strings.Builder{}

	if end > start {
		for _, l := range obj.lines[start:end] {
			s.WriteString(l)
			s.WriteString("\n")
		}
	}

	return strings.TrimRight(s.String(), "\n")
}
