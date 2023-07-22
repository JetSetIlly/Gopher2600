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
	"errors"
	"fmt"
	"io"
	"strings"
)

// callStack maintains information about function calls and the order in which
// they happen.
type callStack struct {
	// call stack of running program
	Stack []*SourceLine

	// list of callers for all executed functions
	Callers map[string]([]*SourceLine)

	// prevLine is helpful when creating the Callers list
	PrevLine *SourceLine
}

// WriteCallstack writes out the current callstack
func (src *Source) WriteCallStack(w io.Writer) {
	for i := 1; i < len(src.CallStack.Stack); i++ {
		w.Write([]byte(src.CallStack.Stack[i].String()))
	}
}

// WriteCallers writes a list of functions that have called the specified function
func (src *Source) WriteCallers(function string, w io.Writer) error {
	callers, ok := src.CallStack.Callers[function]
	if !ok {
		return errors.New(fmt.Sprintf("no function named %s has ever been called", function))
	}

	const maxDepth = 15

	var f func(callLines []*SourceLine, depth int) error
	f = func(callLines []*SourceLine, depth int) error {
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
			if l, ok := src.CallStack.Callers[ln.Function.Name]; ok {
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
