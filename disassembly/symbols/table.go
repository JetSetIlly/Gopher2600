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

package symbols

import (
	"fmt"
	"sort"
	"strings"
)

// table maps a symbol to an address. it also keeps track of the widest symbol
// in the table.
type table struct {
	// indexed by address. addresses should be mapped before indexing takes place
	entries map[uint16]string

	// index of keys in Entries. sortable through the sort.Interface
	idx []uint16

	// the longest symbol in the Entries map
	maxWidth int
}

// newTable is the preferred method of initialisation for the table type.
func newTable() *table {
	t := &table{
		entries: make(map[uint16]string),
		idx:     make([]uint16, 0),
	}
	return t
}

func (t table) String() string {
	s := strings.Builder{}
	for i := range t.idx {
		s.WriteString(fmt.Sprintf("%#04x -> %s\n", t.idx[i], t.entries[t.idx[i]]))
	}
	return s.String()
}

func (t *table) add(addr uint16, symbol string, prefer bool) {
	// end add procedure with check for max symbol width
	defer func() {
		for _, s := range t.entries {
			if len(s) > t.maxWidth {
				t.maxWidth = len(s)
			}
		}
	}()

	// check for duplicates
	for i := range t.idx {
		if t.idx[i] == addr {
			// overwrite existing symbol with preferred symbol
			if prefer {
				t.entries[addr] = symbol
			}
			return
		}
	}

	t.entries[addr] = symbol
	t.idx = append(t.idx, addr)
	sort.Sort(t)
}

func (t table) search(symbol string) (string, uint16, bool) {
	for k, v := range t.entries {
		if strings.ToUpper(v) == symbol {
			return v, k, true
		}
	}
	return "", 0, false
}

// Len implements the sort.Interface.
func (t table) Len() int {
	return len(t.idx)
}

// Less implements the sort.Interface.
func (t table) Less(i, j int) bool {
	return t.idx[i] < t.idx[j]
}

// Swap implements the sort.Interface.
func (t table) Swap(i, j int) {
	t.idx[i], t.idx[j] = t.idx[j], t.idx[i]
}
