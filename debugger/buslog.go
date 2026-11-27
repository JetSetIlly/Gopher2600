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

package debugger

import (
	"bufio"
	"fmt"
	"os"

	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// busLog records the state of the address and data buses after every CPU cycle, for as long as the
// emulation runs. the log is written to a file or, if no filename is given, to the terminal
//
// the log is intended for comparison with the output of a logic analyser attached to a real
// cartridge, so every cycle is recorded, including phantom accesses and cycles where the CPU is
// halted by the RDY line
type busLog struct {
	dbg *Debugger

	// file and buffered writer. both are nil if the log is going to the terminal
	f *os.File
	w *bufio.Writer

	// the number of CPU cycles logged
	cycles int
}

func newBusLog(dbg *Debugger, filename string) (*busLog, error) {
	bl := &busLog{
		dbg: dbg,
	}

	if filename != "" {
		f, err := os.Create(filename)
		if err != nil {
			return nil, fmt.Errorf("bus log: %w", err)
		}
		bl.f = f
		bl.w = bufio.NewWriter(f)
	}

	bl.output(fmt.Sprintf("# bus log for %s", dbg.vcs.Mem.Cart))
	bl.output("# one line for every CPU cycle. the coordinates and the bank are at the end of the cycle")
	bl.output("# columns: cycle, frame, scanline, clock, address, data, driven, access, phantom, area, bank")
	bl.output("# address: as output by the CPU. a cartridge sees only the lower 13 bits (A0 to A12)")
	bl.output("# driven: the data bus pins being driven. pins not driven are floating")
	bl.output("# access: R read, W write, H halted by RDY (no access), J CPU jammed (no access)")
	bl.output("# phantom: P for an access that the CPU makes but ignores")

	return bl, nil
}

func (bl *busLog) output(s string) {
	if bl.w == nil {
		bl.dbg.printLine(terminal.StyleInstrument, "%s", s)
		return
	}
	bl.w.WriteString(s)
	bl.w.WriteByte('\n')
}

// step should be called at the end of every CPU cycle
func (bl *busLog) step() {
	cpu := bl.dbg.vcs.CPU
	mem := bl.dbg.vcs.Mem
	coords := bl.dbg.vcs.TV.GetCoords()

	// the CPU makes no memory access when it is halted or jammed. we can tell because the last
	// instruction is complete but no new instruction has started
	access := "R"
	phantom := "-"
	area := "-"
	bank := "-"
	if cpu.Jammed {
		access = "J"
	} else if cpu.LastResult.Final {
		access = "H"
	} else {
		if mem.LastCPUWrite {
			access = "W"
		}
		if cpu.PhantomMemAccess {
			phantom = "P"
		}
		_, ar := memorymap.MapAddress(mem.AddressBus, !mem.LastCPUWrite)
		area = ar.String()
		if ar == memorymap.Cartridge {
			bank = mem.Cart.GetBank(mem.AddressBus).String()
		}
	}

	bl.output(fmt.Sprintf("%-10d %5d %4d %5d %04x %02x %02x %s %s %-9s %s",
		bl.cycles, coords.Frame, coords.Scanline, coords.Clock,
		mem.AddressBus, mem.DataBus, mem.DataBusDriven,
		access, phantom, area, bank))

	bl.cycles++
}

// end flushes and closes the log file
func (bl *busLog) end() error {
	if bl.w == nil {
		return nil
	}
	if err := bl.w.Flush(); err != nil {
		bl.f.Close()
		return fmt.Errorf("bus log: %w", err)
	}
	if err := bl.f.Close(); err != nil {
		return fmt.Errorf("bus log: %w", err)
	}
	return nil
}
