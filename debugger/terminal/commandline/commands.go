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

package commandline

import (
	"strings"
)

// A commandline Extension provides an instance of Commands such that it can be
// used to extend the number of parameters available to a command, mainly for
// tab-completion purposes
type Extension interface {
	CommandExtension(extension string) *Commands
}

// Commands is the root of the node tree.
type Commands struct {
	index map[string]*node

	// list of commands. should be sorted alphabetically
	list []*node

	// extension handlers. indexed by a name given to the extension in the
	// commands template
	extensions map[string]Extension
}

// Len implements Sort package interface.
func (cmds Commands) Len() int {
	return len(cmds.list)
}

// Less implements Sort package interface.
func (cmds Commands) Less(i int, j int) bool {
	return cmds.list[i].tag < cmds.list[j].tag
}

// Swap implements Sort package interface.
func (cmds Commands) Swap(i int, j int) {
	cmds.list[i], cmds.list[j] = cmds.list[j], cmds.list[i]
}

// String returns the verbose representation of the command tree. Only really
// useful for the validation process.
func (cmds Commands) String() string {
	s := strings.Builder{}
	for c := range cmds.list {
		s.WriteString(cmds.list[c].String())
		s.WriteString("\n")
	}
	return strings.TrimRight(s.String(), "\n")
}

// Usage returns the usage string for a command
func (cmds Commands) Usage(command string) string {
	if c, ok := cmds.index[command]; ok {
		return c.string(true, false)
	}
	return ""
}

func (cmds *Commands) AddExtension(group string, extension Extension) {
	if cmds.extensions == nil {
		cmds.extensions = make(map[string]Extension)
	}
	cmds.extensions[strings.ToLower(group)] = extension
}
