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

package mapfile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type entry struct {
	address      uint32
	functionName string
	objfile      string
	line         int
}

// mapfile contains the parsed information from the map file. this supplements
// the information found in the Source structure and provides a little more
// detail that isn't easily retrieved with just the Source mechanism.
type mapfile struct {
	program []entry
}

const mapFile = "armcode.map"
const mapFile_older = "custom2.map"

func findMapFile(pathToROM string) *os.File {
	// current working directory
	fl, err := os.Open(mapFile)
	if err == nil {
		return fl
	}

	// same direcotry as binary
	fl, err = os.Open(filepath.Join(pathToROM, mapFile))
	if err == nil {
		return fl
	}

	// main sub-directory
	fl, err = os.Open(filepath.Join(pathToROM, "main", mapFile))
	if err == nil {
		return fl
	}

	// main/bin sub-directory
	fl, err = os.Open(filepath.Join(pathToROM, "main", "bin", mapFile))
	if err == nil {
		return fl
	}

	// custom/bin sub-directory. some older DPC+ sources uses this layout
	fl, err = os.Open(filepath.Join(pathToROM, "custom", "bin", mapFile_older))
	if err == nil {
		return fl
	}

	// jetsetilly source tree
	fl, err = os.Open(filepath.Join(pathToROM, "arm", "main.map"))
	if err == nil {
		return fl
	}

	return nil
}

// newMapFile loads and parses a map file. Returns a new instance of Mapfile or
// any errors.
func newMapFile(pathToROM string) (*mapfile, error) {
	mf := &mapfile{
		program: make([]entry, 0, 32),
	}

	// find objdump file and open it
	fl := findMapFile(pathToROM)
	if fl == nil {
		return nil, fmt.Errorf("mapfile: gcc .map file not available (%s)", mapFile)
	}
	defer fl.Close()

	// read all data, split into lines
	data, err := io.ReadAll(fl)
	if err != nil {
		return nil, fmt.Errorf("mapfile: processing error: %w", err)
	}
	lines := strings.Split(string(data), "\n")

	// find the start of mapfile that we're interested in. everything we skip
	// is of no interest or misleading
	for i, l := range lines {
		if l == "Linker script and memory map" {
			lines = lines[i:]
			break // for loop
		}
	}

	var functionName string

	// examine remaining lines of mapfile
	for ln, l := range lines {
		// ignore empty lines
		flds := strings.Fields(l)
		if len(flds) == 0 {
			continue // for loop
		}

		if strings.HasSuffix(l, ".o") {
			// found an .o file, if a function name has been found recently
			// then add a new entry

			if functionName != "" {
				n := strings.LastIndex(l, " ")
				objFile := l[n+1:]

				address, err := strconv.ParseInt(flds[0], 0, 64)
				if err != nil {
					return nil, fmt.Errorf("mapfile: processing error: %w", err)
				}

				mf.program = append(mf.program, entry{
					address:      uint32(address),
					functionName: functionName,
					objfile:      objFile,
					line:         ln,
				})

				functionName = ""
			}

		} else if len(flds) == 1 && strings.HasPrefix(flds[0], ".text.") {
			functionName = flds[0][6:]

			// special condition for main function
			if functionName == "startup.main" {
				functionName = "main"
			}
		}
	}

	return mf, nil
}

// findFunctionName returns the function name for the supplied pc. returns
// the empty string if function name cannot be found.
func (mf *mapfile) findEntry(pc uint32) entry {
	re := entry{}

	for _, e := range mf.program {
		if pc < e.address {
			return re
		}
		re = e
	}

	return re
}
