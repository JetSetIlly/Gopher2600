package symbols

import (
	"fmt"
	"io"
)

// ListSymbols outputs every symbol used in the current ROM
func (tab *Table) ListSymbols(output io.Writer) {
	tab.ListLocations(output)
	tab.ListReadSymbols(output)
	tab.ListWriteSymbols(output)
}

// ListLocations outputs every location symbol used in the current ROM
func (tab *Table) ListLocations(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("Locations\n---------\n")))
	output.Write([]byte(tab.Locations.String()))
}

// ListReadSymbols outputs every read symbol used in the current ROM
func (tab *Table) ListReadSymbols(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("\nRead Symbols\n-----------\n")))
	output.Write([]byte(tab.Read.String()))
}

// ListWriteSymbols outputs every write symbol used in the current ROM
func (tab *Table) ListWriteSymbols(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("\nWrite Symbols\n------------\n")))
	output.Write([]byte(tab.Write.String()))
}
