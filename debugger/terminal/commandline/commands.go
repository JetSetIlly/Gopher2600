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
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
)

// Commands is the root of the node tree.
type Commands struct {
	Index map[string]*node

	cmds []*node

	helpCommand string
	helpCols    int
	helpColFmt  string
	helps       map[string]string
}

// Len implements Sort package interface.
func (cmds Commands) Len() int {
	return len(cmds.cmds)
}

// Less implements Sort package interface.
func (cmds Commands) Less(i int, j int) bool {
	return cmds.cmds[i].tag < cmds.cmds[j].tag
}

// Swap implements Sort package interface.
func (cmds Commands) Swap(i int, j int) {
	cmds.cmds[i], cmds.cmds[j] = cmds.cmds[j], cmds.cmds[i]
}

// String returns the verbose representation of the command tree. Use this only
// for testing/validation purposes. HelpString() is more useful to the end
// user.
func (cmds Commands) String() string {
	s := strings.Builder{}
	for c := range cmds.cmds {
		s.WriteString(cmds.cmds[c].String())
		s.WriteString("\n")
	}
	return strings.TrimRight(s.String(), "\n")
}

// AddHelp adds a "help" command to an already prepared Commands type. it uses
// the top-level nodes of the Commands instance as arguments for the specified
// helpCommand.
func (cmds *Commands) AddHelp(helpCommand string, helps map[string]string) error {
	// if help command exists then there is nothing to do
	if _, ok := cmds.Index[helpCommand]; ok {
		return curated.Errorf("%s: already defined", helpCommand)
	}

	// keep reference to helps
	cmds.helps = helps

	// helpCommand consist of the helpCommand string followed by all the other
	// commands as optional arguments
	defn := strings.Builder{}
	defn.WriteString(helpCommand)
	defn.WriteString(" (")

	// build help command
	longest := 0
	if len(cmds.cmds) > 0 {
		defn.WriteString(cmds.cmds[0].tag)
		for i := 1; i < len(cmds.cmds); i++ {
			defn.WriteString("|")
			if cmds.cmds[i].isPlaceholder() && cmds.cmds[i].placeholderLabel != "" {
				defn.WriteString(cmds.cmds[i].placeholderLabel)
			} else {
				defn.WriteString(cmds.cmds[i].tag)
			}

			if len(cmds.cmds[i].tag) > longest {
				longest = len(cmds.cmds[i].tag)
			}
		}
	}

	// add HELP command itself to list of possible HELP arguments
	defn.WriteString("|")
	defn.WriteString(helpCommand)

	// close argument list
	defn.WriteString(")")

	// parse the constructed definition
	p, d, err := parseDefinition(defn.String(), "")
	if err != nil {
		return curated.Errorf("%s: %s (char %d)", helpCommand, err, d)
	}

	// add parsed definition to list of commands
	cmds.cmds = append(cmds.cmds, p)

	// add to index
	cmds.Index[p.tag] = p

	// record sizing information for help subsystem
	cmds.helpCommand = helpCommand
	cmds.helpCols = 80 / (longest + 3)
	cmds.helpColFmt = fmt.Sprintf("%%%ds", longest+3)

	return nil
}

// HelpOverview returns a columnised list of all help entries.
func (cmds Commands) HelpOverview() string {
	s := strings.Builder{}
	for c := range cmds.cmds {
		s.WriteString(fmt.Sprintf(cmds.helpColFmt, cmds.cmds[c].tag))
		if c%cmds.helpCols == cmds.helpCols-1 {
			s.WriteString("\n")
		}
	}
	return strings.TrimRight(s.String(), "\n")
}

// Help returns the help (and usage for the command).
func (cmds Commands) Help(keyword string) string {
	keyword = strings.ToUpper(keyword)

	s := strings.Builder{}

	if helpTxt, ok := cmds.helps[keyword]; !ok {
		s.WriteString(fmt.Sprintf("no help for %s", keyword))
	} else {
		s.WriteString(helpTxt)
		if cmd, ok := cmds.Index[keyword]; ok {
			s.WriteString("\n\n  Usage: ")
			s.WriteString(cmd.usageString())
		}
	}

	return s.String()
}
