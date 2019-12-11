package symbols

import (
	"fmt"
	"io"
)

// ListSymbols outputs every symbol used in the current ROM
func (tbl *Table) ListSymbols(output io.Writer) {
	tbl.ListLocations(output)
	tbl.ListReadSymbols(output)
	tbl.ListWriteSymbols(output)
}

// ListLocations outputs every location symbol used in the current ROM
func (tbl *Table) ListLocations(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("Locations\n---------\n")))
	output.Write([]byte(tbl.Locations.String()))
}

// ListReadSymbols outputs every read symbol used in the current ROM
func (tbl *Table) ListReadSymbols(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("\nRead Symbols\n-----------\n")))
	output.Write([]byte(tbl.Read.String()))
}

// ListWriteSymbols outputs every write symbol used in the current ROM
func (tbl *Table) ListWriteSymbols(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("\nWrite Symbols\n------------\n")))
	output.Write([]byte(tbl.Write.String()))
}
