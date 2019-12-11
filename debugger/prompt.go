package debugger

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"strings"
)

func (dbg *Debugger) buildPrompt(videoCycle bool) terminal.Prompt {
	// decide which address value to use
	var promptAddress uint16
	var promptBank int

	if dbg.vcs.CPU.LastResult.Final {
		promptAddress = dbg.vcs.CPU.PC.Address()
	} else {
		// if we're in the middle of an instruction then use the
		// addresss in lastResult - in video-stepping mode we want the
		// prompt to report the instruction that we're working on, not
		// the next one to be stepped into.
		promptAddress = dbg.vcs.CPU.LastResult.Address
	}

	promptBank = dbg.vcs.Mem.Cart.GetBank(promptAddress)

	prompt := strings.Builder{}
	prompt.WriteString("[")

	if dbg.scriptScribe.IsActive() {
		prompt.WriteString("(rec)")
	}

	if entry, ok := dbg.disasm.Get(promptBank, promptAddress); ok {
		// because we're using the raw disassmebly the reported address
		// in that disassembly may be misleading.
		prompt.WriteString(fmt.Sprintf(" %#04x %s ]", promptAddress, entry.String()))
	} else {
		// incomplete disassembly, prepare witchspace prompt
		prompt.WriteString(fmt.Sprintf(" %#04x (%d) witchspace ]", promptAddress, promptBank))
	}

	// display indicator that the CPU is waiting for WSYNC to end. only applies
	// when in video step mode.
	if videoCycle && !dbg.vcs.CPU.RdyFlg {
		prompt.WriteString(" ! ")
	}

	// video cycle prompt
	if videoCycle && !dbg.vcs.CPU.LastResult.Final {
		prompt.WriteString(" > ")
		return terminal.Prompt{Content: prompt.String(), Style: terminal.StylePromptVideoStep}
	}

	// cpu cycle prompt
	prompt.WriteString(" >> ")
	return terminal.Prompt{Content: prompt.String(), Style: terminal.StylePromptCPUStep}
}
