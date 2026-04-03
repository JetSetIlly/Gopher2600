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

package symbols

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// load symbols from dasm .sym file if it exists
func (sym *Symbols) fromDasm(cart *cartridge.Cartridge) error {
	// if this is the empty cartridge then this error is expected. return
	// the empty symbol table
	if cart.Filename == "" {
		return nil
	}

	// try to open symbols file
	symFilename := cart.Filename
	ext := filepath.Ext(symFilename)

	// try to figure out the case of the file extension
	if ext == ".BIN" {
		symFilename = fmt.Sprintf("%s.SYM", symFilename[:len(symFilename)-len(ext)])
	} else {
		symFilename = fmt.Sprintf("%s.sym", symFilename[:len(symFilename)-len(ext)])
	}

	sf, err := os.Open(symFilename)
	if err != nil {
		logger.Logf(logger.Allow, "symbols", "dasm .sym file not available (%s)", cart.Filename)
		return nil
	}
	defer sf.Close()

	data, err := io.ReadAll(sf)
	if err != nil {
		return fmt.Errorf("dasm: processing error: %w", err)
	}
	lines := strings.SplitSeq(string(data), "\n")

	// find interesting lines in the symbols file and add to the Symbols
	// instance.
	for ln := range lines {
		// ignore unintersting lines
		if !strings.HasSuffix(ln, "(R )") {
			continue // for loop
		}

		p := strings.Fields(ln)
		if len(p) < 2 || p[0] == "---" {
			continue // for loop
		}

		// get address
		a, err := strconv.ParseUint(p[1], 16, 16)
		if err != nil {
			continue // for loop
		}
		address := uint16(a)

		// add symbol to label list or read/write list

		// get symbol and mapped address and memory area
		symbol := p[0]
		ma, area := memorymap.MapAddress(address, true)

		// remove leading digits if they are present. these digits have
		// been added by DASM for the symbols file
		sp := strings.SplitN(symbol, ".", 2)
		if len(sp) == 2 {
			symbol = symbol[len(sp[0]):]
		}

		switch area {
		case memorymap.Cartridge:
			// adding label for address in every bank for now
			// !!TODO: more selective adding of label from symbols file
			for b := range sym.label {
				sym.label[b].add(SourceDASM, ma, symbol)
			}

		case memorymap.RAM:
			sym.read.add(SourceDASM, address, symbol)
			sym.write.add(SourceDASM, address, symbol)

		default:
			// we do no allow the symbol file to define symbols for other memory areas
		}
	}

	return nil
}
