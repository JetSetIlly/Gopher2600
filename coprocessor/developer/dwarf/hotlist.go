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

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources"
)

// Hotlist can be used to more closely monitor specific global variables
type Hotlist struct {
	src       *Source
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
	h.SaveProject()
}

func (h *Hotlist) add(varb *SourceVariable) {
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

func (h *Hotlist) Add(varb *SourceVariable) {
	h.add(varb)
	h.SaveProject()
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

	h.SaveProject()
}

// file format of hotlist project file
//
// 1) variables separated by newlines
// 2) if a variable is a child of another variable the parent will be listed after "<-"
// 3) the path of parent variables will continue until a variable has no parent

const (
	hotlistRecordSep = "\n"
	hotlistFieldSep  = "<-"
)

func (h *Hotlist) SaveProject() {
	pth, err := resources.JoinPath("developer", h.src.projectID(), "hotlist")
	if err != nil {
		logger.Log(logger.Allow, "dwarf", err.Error())
		return
	}

	f, err := os.Create(pth)
	if err != nil {
		logger.Log(logger.Allow, "dwarf", err.Error())
		return
	}

	for _, v := range h.byAddress {
		f.WriteString(v.Name)

		// record variable path
		for v.parent != nil {
			v = v.parent
			fmt.Fprintf(f, "%s%s", hotlistFieldSep, v.Name)
		}

		f.WriteString(hotlistRecordSep)
	}

	err = f.Close()
	if err != nil {
		logger.Log(logger.Allow, "dwarf", err.Error())
		return
	}
}

func (h *Hotlist) LoadProject() {
	pth, err := resources.JoinLoadPath("developer", h.src.projectID(), "hotlist")
	if err != nil {
		logger.Log(logger.Allow, "dwarf", err.Error())
		return
	}
	f, err := os.Open(pth)
	if err != nil {
		logger.Log(logger.Allow, "dwarf", err.Error())
		return
	}
	defer func() {
		err = f.Close()
		if err != nil {
			logger.Log(logger.Allow, "dwarf", err.Error())
			return
		}
	}()

	d, err := io.ReadAll(f)
	if err != nil {
		logger.Log(logger.Allow, "dwarf", err.Error())
		return
	}

	// not worrying about possibility of \r\n

	for s := range strings.SplitSeq(string(d), hotlistRecordSep) {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			r := strings.Split(s, hotlistFieldSep)

			// the variable path was saved in an order reversed to what we might expect
			slices.Reverse(r)

			// first entry in reversed record will be the base parent (or the actual variable)
			v, ok := h.src.GlobalsByName[r[0]]
			if !ok {
				logger.Logf(logger.Allow, "dwarf", "hotlist: dropped %s: not in current source", r[0])
				continue
			}

			// traverse the variable path
			for _, s := range r[1:] {
				var found bool
				for _, c := range v.children {
					if s == c.Name {
						found = true
						v = c
						break
					}
				}
				if !found {
					v = nil
					break
				}
			}

			if v != nil {
				h.add(v)
			} else {
				logger.Logf(logger.Allow, "dwarf", "hotlist: dropped %s: not in current source", r[len(r)-1])
			}
		}
	}
}
