package input

import (
	"fmt"
	"gopher2600/errors"
	"strings"
)

// ValidateInput checks whether input is correct according to the
// command definitions
func (options Commands) ValidateInput(newInput *Tokens) error {
	var args commandArgList

	tokens := newInput.tokens

	// if tokens is empty then return
	if len(tokens) == 0 {
		return errors.NewGopherError(errors.InputEmpty)
	}

	tokens[0] = strings.ToUpper(tokens[0])

	// basic check for whether command is recognised
	var ok bool
	if args, ok = options[tokens[0]]; !ok {
		return errors.NewGopherError(errors.InputInvalidCommand, fmt.Sprintf("%s is not a debugging command", tokens[0]))
	}

	//  too *many* arguments have been supplied
	if len(tokens)-1 > args.maximumLen() {
		return errors.NewGopherError(errors.InputTooManyArgs, fmt.Sprintf("too many arguments for %s", tokens[0]))
	}

	// too *few* arguments have been supplied
	if len(tokens)-1 < args.requiredLen() {
		switch args[len(tokens)-1].typ {
		case argKeyword:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("keyword required for %s", tokens[0]))
		case argFile:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("filename required for %s", tokens[0]))
		case argValue:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("numeric argument required for %s", tokens[0]))
		case argString:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("string argument required for %s", tokens[0]))
		default:
			// TODO: argument types can be OR'd together. breakdown these types
			// to give more useful information
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("too few arguments for %s", tokens[0]))
		}
	}

	return nil
}
