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

package symbols_test

import (
	_ "embed"
	"os"
	"strings"
	"testing"

	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/test"
)

//go:embed "testdata/expectedDefaultSymbols"
var expectedDefaultSymbols string

func TestDefaultSymbols(t *testing.T) {
	var sym symbols.Symbols

	cart := cartridge.NewCartridge(nil)
	err := sym.ReadDASMSymbolsFile(cart)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}

	w := &strings.Builder{}
	sym.ListSymbols(w)

	if !test.ExpectEquality(t, w.String(), expectedDefaultSymbols) {
		sym.ListSymbols(os.Stdout)
		t.Errorf("default symbols list is wrong")
	}
}

//go:embed "testdata/expectedFlappySymbols"
var expectedFlappySymbols string

func TestFlappySymbols(t *testing.T) {
	var sym symbols.Symbols

	// make a dummy cartridge with the minimum amount of information required
	// for ReadDASMSymbolsFile() to work - the filename of the cartridge is used to
	// identify the symbols file, nothing else is required
	cart := cartridge.NewCartridge(nil)
	cart.Filename = "testdata/flappy.bin"

	err := sym.ReadDASMSymbolsFile(cart)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}

	w := &strings.Builder{}
	sym.ListSymbols(w)

	if !test.ExpectEquality(t, w.String(), expectedFlappySymbols) {
		sym.ListSymbols(os.Stdout)
		t.Errorf("flappy symbols list is wrong")
	}
}
