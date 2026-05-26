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

package dwarf

type Hotlist struct {
	byAddress map[uint64]*SourceVariable
	Sorted    SortedVariables
}

func (h *Hotlist) Len() int {
	return len(h.byAddress)
}

func (h *Hotlist) In(varb *SourceVariable) bool {
	if addr, ok := varb.Address(); ok {
		_, ok := h.byAddress[addr]
		return ok
	}
	return false
}

func (h *Hotlist) Clear() {
	clear(h.byAddress)
	h.Sorted.Variables = h.Sorted.Variables[:0]
}

func (h *Hotlist) Add(varb *SourceVariable) {
	if addr, ok := varb.Address(); ok {
		if _, ok := h.byAddress[addr]; !ok {
			h.byAddress[addr] = varb
			h.Sorted.Variables = append(h.Sorted.Variables, varb)
		}
	}

	h.Sorted.Sort(h.Sorted.method, h.Sorted.descending)

	if len(h.byAddress) != len(h.Sorted.Variables) {
		panic("coprocessor developer: hotlist field are inconsistent")
	}
}

func (h *Hotlist) Remove(varb *SourceVariable) {
	if addr, ok := varb.Address(); ok {
		if _, ok := h.byAddress[addr]; ok {
			delete(h.byAddress, addr)

			// remove from sorted list
			result := h.Sorted.Variables[:0]
			for _, v := range h.Sorted.Variables {
				if vaddr, ok := v.Address(); ok {
					if addr != vaddr {
						result = append(result, v)
					}
				}
			}
			h.Sorted.Variables = result
		}
	}

	if len(h.byAddress) != len(h.Sorted.Variables) {
		panic("coprocessor developer: hotlist field are inconsistent")
	}
}
