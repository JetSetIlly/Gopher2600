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

package dwarf

import (
	"debug/dwarf"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/jetsetilly/gopher2600/logger"
)

func addInstructionsToLines(src *Source, bld *build, symbols []elf.Symbol) error {
	var addressConflicts int

	// we use the symbol table to decide what the end address for a line entry
	// should be if the end of the line reader has been reached
	symAddr := make([]uint64, 0, len(symbols))
	for _, sym := range symbols {
		if sym.Value > 0 && sym.Size > 0 {
			symAddr = append(symAddr, sym.Value)
		}
	}
	sort.Slice(symAddr, func(i, j int) bool {
		return symAddr[i] < symAddr[j]
	})

	// this function uses the symbol table to determine the end address of a line
	// entry. if this is not possible, an address four bytes following the quoted
	// address is returned
	assumeEndAddr := func(addr uint64) uint64 {
		idx := sort.Search(len(symAddr), func(i int) bool {
			return symAddr[i] > addr
		})
		if idx < len(symAddr) {
			return symAddr[idx]
		}
		return addr + 4
	}

	for _, u := range bld.units {
		// read every line in the compile unit
		r, err := bld.dwrf.LineReader(u.e)
		if err != nil {
			return err
		}

		var le dwarf.LineEntry
		for {
			err := r.Next(&le)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break // line entry for loop. will continue with compile unit loop
				}
				return err
			}

			if le.EndSequence {
				continue
			}

			// start and end address of line entry
			var startAddr, endAddr uint64
			startAddr = le.Address

			// end address determined by peeking at the next entry
			p := r.Tell()
			var peek dwarf.LineEntry
			err = r.Next(&peek)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					return err
				}
				endAddr = assumeEndAddr(startAddr)
			} else {
				endAddr = peek.Address
			}
			r.Seek(p)

			// check that the source file has been loaded
			if src.Files[le.File.Name] == nil {
				logger.Logf(logger.Allow, "dwarf", "file not available for linereader: %s", le.File.Name)
				break // line entry for loop. will continue with compile unit loop
			}
			if le.Line == 0 || le.Line-1 > src.Files[le.File.Name].Content.Len() {
				logger.Logf(logger.Allow, "dwarf", "current source is unrelated to ELF/DWARF data (number of lines)")
				break // line entry for loop. will continue with compile unit loop
			}

			ln := src.Files[le.File.Name].Content.Lines[le.Line-1]

			// sanity check start/end address
			if startAddr > endAddr {
				return fmt.Errorf("dwarf: allocate source line: start address (%08x) is after end address (%08x)", startAddr, endAddr)
			}

			// add instruction information to the source line
			if ln != nil {
				// add instruction to source line and add source line to linesByAddress
				for addr := startAddr; addr < endAddr; addr++ {
					// look for address in list of source instructions
					if ins, ok := src.instructions[addr]; ok {
						// add instruction to the list for the source line
						ln.Instruction = append(ln.Instruction, ins)

						// link source line to instruction
						ins.Line = ln

						// add source line to list of lines by address if the
						// address has not been allocated a line already
						if x := src.LinesByAddress[addr]; x == nil {
							src.LinesByAddress[addr] = ln
						} else {
							addressConflicts++
						}

						// advance address value by opcode size. reduce value by
						// one because the loop increment advances by one
						// already (which will always apply even if there is no
						// instruction for the address)
						addr += uint64(ins.size) - 1
					}
				}
			}
		}
	}

	// process again and this time add the breakpoint addresses
	for _, u := range bld.units {
		// read every line in the compile unit
		r, err := bld.dwrf.LineReader(u.e)
		if err != nil {
			return err
		}

		var le dwarf.LineEntry
		for {
			err := r.Next(&le)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break // line entry for loop. will continue with compile unit loop
				}
				return err
			}

			// no need to check whether source file has been loaded because
			// we've already checked that on the previous LineReader run

			// add breakpoint address to the correct line
			if le.IsStmt {
				addr := le.Address
				ln := src.LinesByAddress[addr]
				if ln != nil {
					ln.BreakAddresses = append(ln.BreakAddresses, uint32(addr))
				}
			}
		}
	}

	if addressConflicts > 0 {
		logger.Logf(logger.Allow, "dwarf", "address conflicts when allocating to source lines: %d", addressConflicts)
	}

	return nil
}

func assignFunctionsToLines(src *Source) {
	minRngs := make(map[uint64]SourceRange)

	for _, fn := range src.Functions {
		for _, r := range fn.Range {
			for addr := r.Start; addr <= r.End; addr++ {
				ins, ok := src.instructions[addr]
				if !ok || ins.Line == nil {
					continue
				}

				prev := minRngs[addr]

				// keep previous function or use the new better function
				better := false
				switch {
				case prev.Size() == 0:
					better = true
				case r.Inline && !prev.Inline:
					better = true // inlined takes priority
				case r.Inline == prev.Inline && r.Size() < prev.Size():
					better = true // prefer smaller when same inline status
				}

				if better {
					minRngs[addr] = r
					ins.Line.Function = fn
				}
			}
		}
	}
}
