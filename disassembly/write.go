package disassembly

import (
	"fmt"
	"gopher2600/disassembly/display"
	"gopher2600/errors"
	"io"
)

// Write the entire disassembly to io.Writer
func (dsm *Disassembly) Write(output io.Writer, byteCode bool) error {
	var err error
	for bank := 0; bank < len(dsm.flow); bank++ {
		err = dsm.WriteBank(output, byteCode, bank)
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteBank writes the disassembly of the selected bank to io.Writer
func (dsm *Disassembly) WriteBank(output io.Writer, byteCode bool, bank int) error {
	if bank < 0 || bank > len(dsm.flow)-1 {
		return errors.New(errors.DisasmError, fmt.Sprintf("no such bank (%d)", bank))
	}

	output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))

	for i := range dsm.flow[bank] {
		if d := dsm.flow[bank][i]; d != nil {
			dsm.WriteLine(output, byteCode, d)
		}
	}

	return nil
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
