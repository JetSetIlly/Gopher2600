package input

import (
	"fmt"
	"strings"
)

// CommandTemplate is the root of the argument "tree"
type CommandTemplate map[string]string

// CompileCommandTemplate creates a new instance of Commands from an instance
// of CommandTemplate. if no help command is required, call the function with
// helpKeyword == ""
func CompileCommandTemplate(template CommandTemplate, helpKeyword string) (Commands, error) {
	var err error

	commands := Commands{}
	for k, v := range template {
		commands[k], err = compileTemplateFragment(v)
		if err != nil {
			return nil, fmt.Errorf("error compiling %s: %s", k, err)
		}
	}

	if helpKeyword != "" {
		commands[helpKeyword] = commandArgList{commandArg{typ: argKeyword, required: false, values: &commands}}
	}

	return commands, nil
}

func compileTemplateFragment(fragment string) (commandArgList, error) {
	argl := commandArgList{}

	placeholder := false

	// loop over template string
	for i := 0; i < len(fragment); i++ {
		switch fragment[i] {
		case '%':
			placeholder = true

		case '[':
			// find end of option list
			j := strings.LastIndex(fragment[i:], "]") + i
			if j == -1 {
				return nil, fmt.Errorf("unterminated option list")
			}

			// check for empty list
			if i+1 == j {
				return nil, fmt.Errorf("empty option list")
			}

			// split options list into individual options
			options := strings.Split(fragment[i+1:j], "|")
			if len(options) == 1 {
				options = make([]string, 1)
				options[0] = fragment[i+1 : j]
			}

			// decide whether the option is a required option - if there is an
			// empty option then the option isn't required
			req := true
			for o := 0; o < len(options); o++ {
				if options[o] == "" {
					if req == false {
						return nil, fmt.Errorf("option list can contain only one empty option")
					}
					req = false
				}

				optionParts := strings.Split(options[o], " ")
				if len(optionParts) > 1 {
					return nil, fmt.Errorf("option list can only contain single keywords (%s)", options[o])
				}
			}

			argl = append(argl, commandArg{typ: argKeyword, required: req, values: options})
			i = j

		case ' ':
			// skip spaces

		default:
			if placeholder {
				switch fragment[i] {
				case 'F':
					argl = append(argl, commandArg{typ: argFile, required: true})
				case 'S':
					argl = append(argl, commandArg{typ: argString, required: true})
				case 'V':
					argl = append(argl, commandArg{typ: argValue, required: true})
				case '*':
					argl = append(argl, commandArg{typ: argIndeterminate, required: false})
				default:
					return nil, fmt.Errorf("unknown placeholder directive (%c)", fragment[i])
				}
				placeholder = false
				i++
			} else {
				return nil, fmt.Errorf("unparsable fragment (%s)", fragment, fragment[i])
			}
		}
	}

	return argl, nil
}
