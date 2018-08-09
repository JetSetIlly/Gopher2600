package parser

import (
	"fmt"
	"gopher2600/errors"
	"strings"
)

// ArgType defines the expected argument type
type ArgType int

// the possible values for ArgType
const (
	ArgKeyword ArgType = 1 << iota
	ArgFile
	ArgTarget
	ArgValue
	ArgString
	ArgAddress
	ArgIndeterminate
)

// Commands is the root of the argument "tree"
type Commands map[string]CommandArgs

// Arg specifies the type and properties of an individual argument
type Arg struct {
	Typ  ArgType
	Req  bool
	Vals AllowedVals
}

// CommandArgs is the list of Args for each command
type CommandArgs []Arg

// AllowedVals can take a number of types, useful for tab completion
type AllowedVals interface{}

// Keywords can be used for specifying a list of keywords
// -- satisfies the AllowedVals interface
type Keywords []string

func (a CommandArgs) maxLen() int {
	if len(a) == 0 {
		return 0
	}
	if a[len(a)-1].Typ == ArgIndeterminate {
		return int(^uint(0) >> 1)
	}
	return len(a)
}

func (a CommandArgs) minLen() (m int) {
	for i := 0; i < len(a); i++ {
		if !a[i].Req {
			return
		}
		m++
	}
	return
}

// ValidateInput checks whether input is correct according to the
// command definitions
func (options Commands) ValidateInput(input []string) error {
	var args CommandArgs

	// if input is empty then return
	if len(input) == 0 {
		return errors.NewGopherError(errors.InputEmpty)
	}

	input[0] = strings.ToUpper(input[0])

	// basic check for whether command is recognised
	var ok bool
	if args, ok = options[input[0]]; !ok {
		return errors.NewGopherError(errors.InputInvalidCommand, fmt.Sprintf("%s is not a debugging command", input[0]))
	}

	//  too *many* arguments have been supplied
	if len(input)-1 > args.maxLen() {
		return errors.NewGopherError(errors.InputTooManyArgs, fmt.Sprintf("too many arguments for %s", input[0]))
	}

	// too *few* arguments have been supplied
	if len(input)-1 < args.minLen() {
		switch args[len(input)-1].Typ {
		case ArgKeyword:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("keyword required for %s", input[0]))
		case ArgFile:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("filename required for %s", input[0]))
		case ArgAddress:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("address required for %s", input[0]))
		case ArgTarget:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("emulation target required for %s", input[0]))
		case ArgValue:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("numeric argument required for %s", input[0]))
		case ArgString:
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("string argument required for %s", input[0]))
		default:
			// TODO: argument types can be OR'd together. breakdown these types
			// to give more useful information
			return errors.NewGopherError(errors.InputTooFewArgs, fmt.Sprintf("too few arguments for %s", input[0]))
		}
	}

	return nil
}
