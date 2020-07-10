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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package debugger

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/disassembly"
)

func (dbg *Debugger) buildPrompt(videoCycle bool) terminal.Prompt {
	prompt := strings.Builder{}
	prompt.WriteString("[")

	if dbg.scriptScribe.IsActive() {
		prompt.WriteString("(rec)")
	}

	// decide which address value to use
	var e *disassembly.Entry
	if dbg.VCS.CPU.LastResult.Final || dbg.VCS.CPU.HasReset() {
		e = dbg.Disasm.GetEntryByAddress(dbg.VCS.CPU.PC.Address())
	} else {
		// if we're in the middle of an instruction then use the addresss in
		// lastResult. in these instances we want the  prompt to report the
		// instruction that the CPU is working on, not the next one to be
		// stepped into.
		e = dbg.lastResult
	}

	// build prompt based on how confident we are of the contents of the
	// disassembly entry. starting with the condition of no disassembly at all
	if e == nil {
		prompt.WriteString(" unsure")
	} else if e.Level == disassembly.EntryLevelUnused {
		prompt.WriteString(fmt.Sprintf(" %s unsure", e.Address))
	} else {
		prompt.WriteString(fmt.Sprintf(" %s %s", e.Address, e.Mnemonic))
		if e.Operand != "" {
			prompt.WriteString(fmt.Sprintf(" %s", e.Operand))
		}
	}
	prompt.WriteString(" ]")

	// display indicator that the CPU is waiting for WSYNC to end. only applies
	// when in video step mode.
	if videoCycle && !dbg.VCS.CPU.RdyFlg {
		prompt.WriteString(" !")
	}

	// video cycle prompt
	if !dbg.VCS.CPU.LastResult.Final {
		if videoCycle {
			prompt.WriteString(" > ")
		} else {
			// we're in the middle of a cpu instruction but this is not a video
			// cycle prompt. while this is possible it is unusual. indicate
			// this by appending a double question mark
			//
			// an example of this is when the supercharger.TapeLoaded error has
			// been triggered
			prompt.WriteString(" ?? ")
		}
		return terminal.Prompt{Content: prompt.String(), Style: terminal.StylePromptVideoStep}
	}

	// cpu cycle prompt
	prompt.WriteString(" >> ")
	return terminal.Prompt{Content: prompt.String(), Style: terminal.StylePromptCPUStep}
}
