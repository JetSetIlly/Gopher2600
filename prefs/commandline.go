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

package prefs

import (
	"fmt"
	"sort"
	"strings"
)

var commandLineStack []map[string]Value

func init() {
	commandLineStack = make([]map[string]Value, 0)
}

// SizeCommandLineStack returns the number of groups that have been added with
// AddCommanLineGroup().
func SizeCommandLineStack() int {
	return len(commandLineStack)
}

// PopCommandLineStack forgets the most recent group added by
// AddCommandLineGroup().
//
// Returns the "unused" preferences of the stack entry.
func PopCommandLineStack() string {
	if len(commandLineStack) == 0 {
		return ""
	}

	// get top of stack
	popped := commandLineStack[len(commandLineStack)-1]

	// remove the top of the stack
	commandLineStack = commandLineStack[:len(commandLineStack)-1]

	// rebuild the prefs string from the remaining entries from the old stack top
	keys := make([]string, 0, len(popped))
	for key := range popped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	s := strings.Builder{}
	for _, key := range keys {
		s.WriteString(fmt.Sprintf("%s::%v; ", key, popped[key]))
	}

	// return prefs string
	return strings.TrimSuffix(s.String(), "; ")
}

// PushCommandLineStack parses a command line and adds it as a new group.
func PushCommandLineStack(prefs string) {
	commandLineStack = append(commandLineStack, make(map[string]Value))
	cl := commandLineStack[len(commandLineStack)-1]

	// divide prefs string into individual key/value pairs
	o := strings.Split(prefs, ";")

	for _, p := range o {
		// split key/value
		kv := strings.Split(p, "::")

		// add to top of stack
		if len(kv) == 2 {
			cl[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
}

// GetCommandLinePref value from current group. The value is deleted when it is returned.
func GetCommandLinePref(key string) (bool, Value) {
	if len(commandLineStack) == 0 {
		return false, nil
	}

	// top of stack
	cl := commandLineStack[len(commandLineStack)-1]

	// return value for key if present. delete that entry.
	if v, ok := cl[key]; ok {
		delete(cl, key)
		return true, v
	}

	return false, nil
}
