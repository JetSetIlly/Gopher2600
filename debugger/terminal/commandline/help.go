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
	"sort"
	"strings"
)

// The command that should be used to invoke the HELP system
const HelpCommand = "HELP"

// AddHelp() adds the HELP command to the list of commands if it hasn't been
// added already (or was part of the template given to the ParseCommandTemplate()
// function)
//
// It is up to the user of the commandline package to do something with the HELP
// command
func AddHelp(cmds *Commands) error {
	// add help command only if it's not been added already
	if _, ok := cmds.index[HelpCommand]; ok {
		return nil
	}

	// create definition string using existing command list
	var def string
	def = fmt.Sprintf("%s (", HelpCommand)
	for _, n := range cmds.list {
		def = fmt.Sprintf("%s%s|", def, n.tag)
	}
	def = strings.TrimSuffix(def, "|")
	def = fmt.Sprintf("%s)", def)

	// parse definition and add it to the list of commands
	h, _, err := parseDefinition(def, "")
	if err != nil {
		return fmt.Errorf("parser: error adding HELP command: %w", err)
	}
	cmds.index[HelpCommand] = h
	cmds.list = append(cmds.list, h)

	// resort changed list of commands
	sort.Stable(cmds)

	return nil
}

// HelpSummary returns a string showing the top-level HELP topics in five columns
func HelpSummary(cmds *Commands) string {
	// which help topic is the longest (string length). we should maybe do this
	// once after adding the commands but this is simpler and with no impact
	var longest int
	for _, n := range cmds.list {
		longest = max(longest, len(n.tag))
	}
	longest += 2

	var s strings.Builder
	for i, n := range cmds.list {
		s.WriteString(n.tag)
		if (i+1)%5 == 0 {
			s.WriteString("\n")
		} else {
			s.WriteString(strings.Repeat(" ", longest-len(n.tag)))
		}
	}
	return s.String()
}
