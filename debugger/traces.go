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
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
)

type tracer struct {
	ai addressInfo
}

func (t tracer) String() string {
	return t.ai.String()
}

// the list of currently defined traces in the system.
type traces struct {
	dbg                 *Debugger
	traces              []tracer
	lastAddressAccessed uint16
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
		return curated.Errorf("trace #%d is not defined", num)
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

	s := strings.Builder{}

	for i := range trc.traces {
		// continue loop if we're not matching last address accessed
		if trc.traces[i].ai.address != trc.dbg.VCS.Mem.LastAccessAddress {
			continue
		}

		// continue if this is a repeat of the last address accessed
		if trc.lastAddressAccessed == trc.dbg.VCS.Mem.LastAccessAddress {
			continue
		}

		if trc.dbg.VCS.Mem.LastAccessWrite {
			s.WriteString("write ")
		} else {
			s.WriteString("read ")
		}

		s.WriteString(trc.traces[i].String())
		break // for loop
	}

	// note what the last address accessed was
	trc.lastAddressAccessed = trc.dbg.VCS.Mem.LastAccessAddress

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
	// get address. required.
	a, _ := tokens.Get()

	// convert address
	var ai *addressInfo
	ai = trc.dbg.dbgmem.mapAddress(a, true)
	if ai == nil {
		ai = trc.dbg.dbgmem.mapAddress(a, false)
		if ai == nil {
			return curated.Errorf("invalid trace address: %s", a)
		}
	}

	nt := tracer{ai: *ai}

	// check to see if trace already exists
	for _, t := range trc.traces {
		if t.ai.address == nt.ai.address {
			return curated.Errorf("already being traced (%s)", t)
		}
	}

	// add trace
	trc.traces = append(trc.traces, nt)

	return nil
}
