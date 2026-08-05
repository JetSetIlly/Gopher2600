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
	"os"

	"github.com/jetsetilly/gopher2600/resources/unique"
)

// save all SortedVariables to a CSV file in the working directory. filename will be of the form:
//
// <name>_<cart name>_<timestamp>.csv
//
// all entries in the supplied SortedVariables are saved, including closed nodes.
func SaveToCSV(varbs SortedVariables, name string, cartName string) (fn string, rerr error) {
	// open unique file
	fn = fmt.Sprintf("%s.csv", unique.Filename(name, cartName))
	f, err := os.Create(fn)
	if err != nil {
		return "", fmt.Errorf("could not save globals CSV: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			fn = ""
			rerr = fmt.Errorf("could not save globals CSV: %w", err)
		}
	}()

	// write variable to file
	writeVarb := func(varb *SourceVariable) {
		fmt.Fprintf(f, "%s,", varb.Name)
		fmt.Fprintf(f, "%s,", varb.Type.Name)
		if a, ok := varb.Address(); ok {
			fmt.Fprintf(f, "%08x,", a)
		} else {
			f.WriteString(",")
		}

		fmt.Fprintf(f, varb.Type.Hex(), varb.Value())
		f.WriteString("\n")
	}

	// the builEntry function is recursive and will is very similar in
	// structure to the drawVariable() function above
	var buildEntry func(*SourceVariable, string)
	buildEntry = func(varb *SourceVariable, parent string) {
		fmt.Fprintf(f, "%s,", parent)

		// how we write the line differs depending on whether the variable has
		// children or not
		if varb.NumChildren() > 0 {
			if parent != "" {
				parent = fmt.Sprintf("%s->%s", parent, varb.Name)
			} else {
				parent = varb.Name
				writeVarb(varb)
			}

			for i := 0; i < varb.NumChildren(); i++ {
				buildEntry(varb.Child(i), parent)
			}
		} else {
			writeVarb(varb)
		}
	}

	// write header to CSV file
	f.WriteString("Parent, Name, Type, Address, Value\n")

	// process every variable in the current view
	for _, v := range varbs.Variables {
		buildEntry(v, "")
	}

	return fn, nil
}
