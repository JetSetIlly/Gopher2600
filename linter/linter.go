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

package linter

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/disassembly"
)

// Lint disassembly and return results.
func Lint(dsm *disassembly.Disassembly) (*Results, error) {
	res := newResults()

	err := dsm.IterateBlessed(nil, func(e *disassembly.Entry) string {
		res.lint(e)
		return ""
	})

	return res, err
}

// LintEntry for every lint error detected in disassembly entry.
type LintEntry struct {
	DisasmEntry *disassembly.Entry
	Error       string
	Details     interface{}
}

func (le *LintEntry) String() string {
	s := strings.Builder{}
	s.WriteString(le.DisasmEntry.StringColumnated(disassembly.ColumnAttr{}))
	s.WriteString(fmt.Sprintf("\t%s", le.Error))
	switch le.Details.(type) {
	case string:
		s.WriteString(fmt.Sprintf(" [%s]", le.Details))
	case uint8:
		s.WriteString(fmt.Sprintf(" [$%02x]", le.Details))
	case uint16:
		s.WriteString(fmt.Sprintf(" [$%04x]", le.Details))
	}
	return s.String()
}

// Results is a list of LintEntries grouped by Error.
type Results map[string][]*LintEntry

func newResults() *Results {
	res := make(Results)
	return &res
}

func (res *Results) String() string {
	s := strings.Builder{}
	for _, r := range *res {
		for _, e := range r {
			s.WriteString(e.String())
			s.WriteString("\n")
		}
	}
	return s.String()
}

func (res *Results) lint(e *disassembly.Entry) {
	for _, le := range rules(e) {
		if _, ok := (*res)[le.Error]; !ok {
			(*res)[le.Error] = make([]*LintEntry, 0)
		}
		(*res)[le.Error] = append((*res)[le.Error], le)
	}
}
