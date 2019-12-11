package commandline

import (
	"gopher2600/errors"
	"strings"
)

// AddHelp adds a "help" command to an already prepared Commands type. it uses
// the top-level nodes of the Commands instance as arguments for the specified
// helpCommand
func (cmds *Commands) AddHelp(helpCommand string) error {

	// iterate through command tree and return error if HELP command is already
	// defined
	for i := 1; i < len(*cmds); i++ {
		if (*cmds)[i].tag == helpCommand {
			return errors.New(errors.ParserError, helpCommand, "already defined", 0)
		}
	}

	// HELP command consist of the helpCommand string followed by all the other
	// commands as optional arguments
	defn := strings.Builder{}
	defn.WriteString(helpCommand)
	defn.WriteString(" (")

	if len(*cmds) > 0 {
		defn.WriteString((*cmds)[0].tag)
		for i := 1; i < len(*cmds); i++ {
			defn.WriteString("|")
			defn.WriteString((*cmds)[i].tag)
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
		return errors.New(errors.ParserError, helpCommand, err, d)
	}

	// add parsed definition to list of commands
	*cmds = append((*cmds), p)

	return nil
}
