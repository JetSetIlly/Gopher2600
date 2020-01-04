package debugger

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"gopher2600/hardware/memory/memorymap"
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

	// build prompt depending on address and whether a disassembly is available
	if promptAddress < memorymap.OriginCart {
		// prompt address doesn't seem to be pointing to the cartridge, prepare
		// "non-cart" prompt
		prompt.WriteString(fmt.Sprintf(" %#04x non-cart space ]", promptAddress))
	} else if d, ok := dbg.disasm.Get(promptBank, promptAddress); ok {
		// because we're using the raw disassmebly the reported address
		// in that disassembly may be misleading.
		prompt.WriteString(fmt.Sprintf(" %#04x %s", promptAddress, d.Mnemonic))
		if d.Operand != "" {
			prompt.WriteString(fmt.Sprintf(" %s", d.Operand))
		}
		prompt.WriteString(" ]")
	} else {
		// incomplete disassembly, prepare "no disasm" prompt
		ai := dbg.dbgmem.mapAddress(promptAddress, true)
		if ai == nil {
			prompt.WriteString(fmt.Sprintf(" %#04x (%d) unmappable address ]", promptAddress, promptBank))
		} else {
			switch ai.area {
			case memorymap.RAM:
				prompt.WriteString(fmt.Sprintf(" %#04x (%d) in RAM! ]", promptAddress, promptBank))
			case memorymap.Cartridge:
				prompt.WriteString(fmt.Sprintf(" %#04x (%d) no disasm ]", promptAddress, promptBank))
			default:
				// if we're not in RAM or Cartridge space then we must be in
				// the TIA or RIOT - this would be very odd indeed
				prompt.WriteString(fmt.Sprintf(" %#04x (%d) WTF! ]", promptAddress, promptBank))
			}
		}
	}

	// display indicator that the CPU is waiting for WSYNC to end. only applies
	// when in video step mode.
	if videoCycle && !dbg.vcs.CPU.RdyFlg {
		prompt.WriteString(" !")
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
