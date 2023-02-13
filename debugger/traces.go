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
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/debugger/dbgmem"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
)

type tracer struct {
	ai dbgmem.AddressInfo

	// whether the address should be interpreted strictly or whether mirrors
	// should be considered too
	strict bool
}

func (t tracer) String() string {
	strict := ""
	if t.strict {
		strict = " (strict)"
	}
	return fmt.Sprintf("%s%s", t.ai, strict)
}

// the list of currently defined traces in the system.
type traces struct {
	dbg                       *Debugger
	traces                    []tracer
	lastAddressAccessedMapped uint16
}

// newTraces is the preferred method of initialisation for the traces type.
func newTraces(dbg *Debugger) *traces {
	trc := &traces{
		dbg: dbg,
	}
	trc.clear()
	return trc
}

// clear all traces.
func (trc *traces) clear() {
	trc.traces = make([]tracer, 0, 10)
}

// drop a specific tracer by a position in the list.
func (trc *traces) drop(num int) error {
	if len(trc.traces)-1 < num {
		return fmt.Errorf("trace #%d is not defined", num)
	}

	h := trc.traces[:num]
	t := trc.traces[num+1:]
	trc.traces = make([]tracer, len(h)+len(t), cap(trc.traces))
	copy(trc.traces, h)
	copy(trc.traces[len(h):], t)

	return nil
}

// check compares the current state of the emulation with every trace
// condition. returns a string with the first match found (it is not believed
// to be possible for more one trace to match at the same time).
func (trc *traces) check() string {
	if len(trc.traces) == 0 {
		return ""
	}

	// no check for phantom access
	if trc.dbg.vcs.CPU.PhantomMemAccess {
		return ""
	}

	// no check if access address hasn't changed.
	//
	// note that unlike watches.check() we don't compare the write flag - we
	// want to trace both types of accesses even if it's on the same address.
	if trc.lastAddressAccessedMapped == trc.dbg.vcs.Mem.LastCPUAddressMapped {
		return ""
	}

	s := strings.Builder{}

	for _, t := range trc.traces {
		// pick which addresses to compare depending on whether watch is strict
		if t.strict {
			if trc.dbg.vcs.Mem.LastCPUAddressLiteral != t.ai.Address {
				continue
			}
		} else {
			if trc.dbg.vcs.Mem.LastCPUAddressMapped != t.ai.MappedAddress {
				continue
			}
		}

		lai := trc.dbg.dbgmem.GetAddressInfo(trc.dbg.vcs.Mem.LastCPUAddressLiteral, !trc.dbg.vcs.Mem.LastCPUWrite)

		if trc.dbg.vcs.Mem.LastCPUWrite {
			s.WriteString(fmt.Sprintf("write %#02x to %s ", trc.dbg.vcs.Mem.LastCPUData, lai))
		} else {
			s.WriteString(fmt.Sprintf("read %#02x from %s ", trc.dbg.vcs.Mem.LastCPUData, lai))
		}

		break // for loop
	}

	// note what the last address accessed was
	trc.lastAddressAccessedMapped = trc.dbg.vcs.Mem.LastCPUAddressMapped

	return s.String()
}

// list currently defined traces.
func (trc *traces) list() {
	if len(trc.traces) == 0 {
		trc.dbg.printLine(terminal.StyleFeedback, "no traces")
	} else {
		trc.dbg.printLine(terminal.StyleFeedback, "traces:")
		for i := range trc.traces {
			trc.dbg.printLine(terminal.StyleFeedback, "% 2d: %s", i, trc.traces[i])
		}
	}
}

// parse tokens and add new trace. only one trace at a time can be specified on
// the command line.
func (trc *traces) parseCommand(tokens *commandline.Tokens) error {
	var strict bool

	// strict addressing or not
	arg, _ := tokens.Get()
	arg = strings.ToUpper(arg)
	switch arg {
	case "STRICT":
		strict = true
	default:
		strict = false
		tokens.Unget()
	}

	// get address. required.
	a, _ := tokens.Get()

	// convert address
	var ai *dbgmem.AddressInfo
	ai = trc.dbg.dbgmem.GetAddressInfo(a, true)
	if ai == nil {
		ai = trc.dbg.dbgmem.GetAddressInfo(a, false)
		if ai == nil {
			return fmt.Errorf("invalid trace address (%s) expecting 16-bit address or symbol", a)
		}
	}

	nt := tracer{ai: *ai, strict: strict}

	// check to see if trace already exists
	for _, t := range trc.traces {
		if t.ai.Address == nt.ai.Address {
			return fmt.Errorf("already being traced (%s)", t)
		}
	}

	// add trace
	trc.traces = append(trc.traces, nt)

	return nil
}
