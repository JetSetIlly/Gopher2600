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

	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/disassembly"
)

func (dbg *Debugger) buildPrompt() terminal.Prompt {
	s := strings.Builder{}

	if dbg.liveBankInfo.ExecutingCoprocessor {
		e := dbg.Disasm.GetEntryByAddress(dbg.liveBankInfo.CoprocessorResumeAddr)
		return terminal.Prompt{
			Content:   fmt.Sprintf("* %s", e.String()),
			Recording: dbg.scriptScribe.IsActive(),
		}
	}

	if dbg.liveBankInfo.Sequential {
		return terminal.Prompt{
			Content:   fmt.Sprintf("$%04x", dbg.vcs.CPU.PC.Address()),
			Recording: dbg.scriptScribe.IsActive(),
		}
	}

	var e *disassembly.Entry

	// decide which address value to use
	if dbg.vcs.CPU.LastResult.Final {
		e = dbg.Disasm.GetEntryByAddress(dbg.vcs.CPU.PC.Address())
	} else {
		// if we're in the middle of an instruction then use the addresss in
		// lastResult. in these instances we want the prompt to report the
		// instruction that the CPU is working on, not the next one to be
		// stepped into.
		e = dbg.liveDisasmEntry
	}

	// build prompt based on how confident we are of the contents of the
	// disassembly entry. starting with the condition of no disassembly at all
	if e == nil {
		s.WriteString(fmt.Sprintf("$%04x", dbg.vcs.CPU.PC.Address()))
	} else if e.Level == disassembly.EntryLevelUnmappable {
		s.WriteString(e.Address)
	} else {
		// this is the ideal path. the address is in the disassembly and we've
		// decoded it already
		s.WriteString(fmt.Sprintf("%s %s", e.Address, e.Operator))

		if e.Operand.Resolve() != "" {
			s.WriteString(fmt.Sprintf(" %s", e.Operand.Resolve()))
		}
	}

	p := terminal.Prompt{
		Content:   s.String(),
		Recording: dbg.scriptScribe.IsActive(),
	}

	if coproc := dbg.vcs.Mem.Cart.GetCoProcBus(); coproc != nil {
		state := coproc.CoProcExecutionState()
		p.CoProcYield = state.Yield
	}

	// LastResult final is false on CPU reset so we must check for that also
	if dbg.vcs.CPU.LastResult.Final {
		p.Type = terminal.PromptTypeCPUStep
	} else {
		p.Type = terminal.PromptTypeVideoStep
	}

	return p
}
