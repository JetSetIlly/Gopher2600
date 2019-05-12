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
	for i := range tab.locations {
		output.Write([]byte(fmt.Sprintf("%#04x -> %s\n", i, tab.Locations[tab.locations[i]])))
	}
}

// ListReadSymbols outputs every read symbol used in the current ROM
func (tab *Table) ListReadSymbols(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("\nRead Symbols\n-----------\n")))
	for i := range tab.readSymbols {
		output.Write([]byte(fmt.Sprintf("%#04x -> %s\n", i, tab.ReadSymbols[tab.readSymbols[i]])))
	}
}

// ListWriteSymbols outputs every write symbol used in the current ROM
func (tab *Table) ListWriteSymbols(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("\nWrite Symbols\n------------\n")))
	for i := range tab.writeSymbols {
		output.Write([]byte(fmt.Sprintf("%#04x -> %s\n", i, tab.WriteSymbols[tab.writeSymbols[i]])))
	}
}
