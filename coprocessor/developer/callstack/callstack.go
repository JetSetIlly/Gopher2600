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

package callstack

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
)

// callStack maintains information about function calls and the order in which
// they happen.
type CallStack struct {
	// call stack of running program
	Stack []*dwarf.SourceLine

	// list of callers for all executed functions
	Callers map[string]([]*dwarf.SourceLine)

	// prevLine is helpful when creating the Callers list
	PrevLine *dwarf.SourceLine
}

// WriteCallstack writes out the current callstack
func (cs *CallStack) WriteCallStack(w io.Writer) {
	for i := 1; i < len(cs.Stack); i++ {
		w.Write([]byte(cs.Stack[i].String()))
	}
}

// WriteCallers writes a list of functions that have called the specified function
func (cs *CallStack) WriteCallers(function string, w io.Writer) error {
	callers, ok := cs.Callers[function]
	if !ok {
		return errors.New(fmt.Sprintf("no function named %s has ever been called", function))
	}

	const maxDepth = 15

	var f func(callLines []*dwarf.SourceLine, depth int) error
	f = func(callLines []*dwarf.SourceLine, depth int) error {
		indent := strings.Builder{}
		for i := 0; i < depth; i++ {
			indent.WriteString("  ")
		}

		if depth > maxDepth {
			return errors.New(fmt.Sprintf("%stoo deep", indent.String()))
		}

		for _, ln := range callLines {
			if ln.IsStub() {
				return nil
			}

			s := fmt.Sprintf("%s (%s:%d)", ln.Function.Name, ln.File.ShortFilename, ln.LineNumber)
			w.Write([]byte(fmt.Sprintf("%s%s", indent.String(), s)))
			if l, ok := cs.Callers[ln.Function.Name]; ok {
				err := f(l, depth+1)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	w.Write([]byte(function))
	return f(callers, 1)
}
