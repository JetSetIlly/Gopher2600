package input

import (
	"fmt"
	"strings"
)

// CommandTemplate is the root of the argument "tree"
type CommandTemplate map[string]string

// CompileCommandTemplate creates a new instance of Commands from an instance
// of CommandTemplate. if no help is command is required, use the empty-string
// to for the helpKeyword argument
func CompileCommandTemplate(template CommandTemplate, helpKeyword string) (Commands, error) {
	commands := Commands{}
	for k, v := range template {
		commands[k] = commandArgList{}

		placeholder := false

		for i := 0; i < len(v); i++ {
			switch v[i] {
			case '%':
				placeholder = true
			case '[':
				// find end of option list
				j := strings.Index(v[i:], "]")
				if j == -1 {
					return commands, fmt.Errorf("unclosed option list (%s)", k)
				}

				options := strings.Split(v[i+1:j], "|")
				if len(options) == 1 {
					// note: Split() returns a slice of the input string, if
					// the seperator ("|") cannot be found. the length of an
					// empty option list is therefore 1.
					return commands, fmt.Errorf("empty option list (%s)", k)
				}

				// decide whether the option is required
				req := true
				for m := 0; m < len(options); m++ {
					if options[m] == "" {
						req = false
						break
					}
				}

				// add a new argument for current keyword with the options
				// we've found
				commands[k] = append(commands[k], commandArg{typ: argKeyword, required: req, values: options})

			default:
				if placeholder {
					switch v[i] {
					case 'F':
						commands[k] = append(commands[k], commandArg{typ: argFile, required: true})
					case 'S':
						commands[k] = append(commands[k], commandArg{typ: argString, required: true})
					case 'V':
						commands[k] = append(commands[k], commandArg{typ: argValue, required: true})
					case '*':
						commands[k] = append(commands[k], commandArg{typ: argIndeterminate, required: true})
					}
				}
			}
		}
	}

	if helpKeyword != "" {
		commands[helpKeyword] = commandArgList{commandArg{typ: argKeyword, required: false, values: &commands}}
	}

	return commands, nil
}
