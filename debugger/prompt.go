package debugger

import (
	"fmt"
	"gopher2600/debugger/console"
	"strings"
)

func (dbg *Debugger) buildPrompt(videoCycle bool) console.Prompt {
	// decide which address value to use
	var promptAddress uint16
	var promptBank int

	if dbg.lastResult == nil || dbg.lastResult.Final {
		promptAddress = dbg.vcs.CPU.PC.ToUint16()
	} else {
		// if we're in the middle of an instruction then use the
		// addresss in lastResult - in video-stepping mode we want the
		// prompt to report the instruction that we're working on, not
		// the next one to be stepped into.
		promptAddress = dbg.lastResult.Address
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
	if videoCycle && !dbg.lastResult.Final {
		prompt.WriteString(" > ")
		return console.Prompt{Content: prompt.String(), Style: console.StylePromptAlt}
	}

	// cpu cycle prompt
	prompt.WriteString(" >> ")
	return console.Prompt{Content: prompt.String(), Style: console.StylePrompt}
}
