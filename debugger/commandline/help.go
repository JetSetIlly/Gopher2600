package commandline

import (
	"gopher2600/errors"
	"strings"
)

// AddHelp adds a "help" command to an already prepared Commands type. it uses
// the top-level nodes of the Commands instance as arguments for the specified
// helpCommand
func (cmds *Commands) AddHelp(helpCommand string) error {
	for i := 1; i < len(*cmds); i++ {
		if (*cmds)[i].tag == helpCommand {
			return errors.NewFormattedError(errors.ParserError, helpCommand, "already present", 0)
		}
	}

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

	defn.WriteString(")")

	p, d, err := parseDefinition(defn.String(), "")
	if err != nil {
		return errors.NewFormattedError(errors.ParserError, helpCommand, err, d)
	}

	*cmds = append((*cmds), p)

	return nil
}
