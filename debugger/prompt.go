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
	content := strings.Builder{}

	var e *disassembly.Entry

	// decide which address value to use
	if dbg.VCS.CPU.LastResult.Final || dbg.VCS.CPU.HasReset() {
		e = dbg.Disasm.GetEntryByAddress(dbg.VCS.CPU.PC.Address())
	} else {
		// if we're in the middle of an instruction then use the addresss in
		// lastResult. in these instances we want the prompt to report the
		// instruction that the CPU is working on, not the next one to be
		// stepped into.
		e = dbg.lastResult
	}

	// build prompt based on how confident we are of the contents of the
	// disassembly entry. starting with the condition of no disassembly at all
	if e == nil {
		content.WriteString(fmt.Sprintf("$%04x", dbg.VCS.CPU.PC.Address()))
	} else if e.Level == disassembly.EntryLevelUnmappable {
		content.WriteString(e.Address)
	} else {
		// this is the ideal path. the address is in the disassembly and we've
		// decoded it already
		content.WriteString(fmt.Sprintf("%s %s", e.Address, e.Mnemonic))

		if e.Operand.String() != "" {
			content.WriteString(fmt.Sprintf(" %s", e.Operand))
		}
	}

	p := terminal.Prompt{
		Content:   content.String(),
		Recording: dbg.scriptScribe.IsActive(),
		CPURdy:    dbg.VCS.CPU.RdyFlg,
	}

	if dbg.VCS.CPU.LastResult.Final {
		p.Type = terminal.PromptTypeCPUStep
	} else {
		p.Type = terminal.PromptTypeVideoStep
	}

	return p
}
