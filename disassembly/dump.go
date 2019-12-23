package disassembly

import (
	"fmt"
	"gopher2600/disassembly/display"
	"io"
)

// Write writes the entire disassembly to io.Writer
func (dsm *Disassembly) Write(output io.Writer, byteCode bool) {
	for bank := 0; bank < len(dsm.flow); bank++ {
		output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))

		for i := range dsm.flow[bank] {
			if d := dsm.flow[bank][i]; d != nil {
				dsm.WriteLine(output, byteCode, d)
			}
		}
	}
}

// WriteLine writes a single Instruction to io.Writer
func (dsm *Disassembly) WriteLine(output io.Writer, byteCode bool, d *display.Instruction) {
	if d.Location != "" {
		output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Location, d.Location)))
		output.Write([]byte("\n"))
	}

	if byteCode {
		output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Bytecode, d.Bytecode)))
		output.Write([]byte(" "))
	}

	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Address, d.Address)))
	output.Write([]byte(" "))
	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Mnemonic, d.Mnemonic)))
	output.Write([]byte(" "))
	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Operand, d.Operand)))
	output.Write([]byte(" "))
	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Cycles, d.Cycles)))
	output.Write([]byte(" "))
	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Notes, d.Notes)))

	output.Write([]byte("\n"))
}
