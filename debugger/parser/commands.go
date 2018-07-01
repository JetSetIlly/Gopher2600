package parser

import (
	"fmt"
)

// ArgType defines the expected argument type
type ArgType int

// the possible values for ArgType
const (
	ArgKeyword ArgType = iota
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
	if len(a) == 1 {
		if a[0].Typ == ArgIndeterminate {
			return int(^uint(0) >> 1)
		}
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

// CheckCommandInput checks whether input is correct according to the
// command definitions
func (options Commands) CheckCommandInput(input []string) error {
	var args CommandArgs

	// basic check for whether command is recognised
	var ok bool
	if args, ok = options[input[0]]; !ok {
		return fmt.Errorf("%s is not a debugging command", input[0])
	}

	//  too *many* arguments have been supplied
	if len(input)-1 > args.maxLen() {
		return fmt.Errorf("too many arguments for %s", input[0])
	}

	// too *few* arguments have been supplied
	if len(input)-1 < args.minLen() {
		switch args[len(input)-1].Typ {
		case ArgKeyword:
			return fmt.Errorf("keyword required for %s", input[0])
		case ArgFile:
			return fmt.Errorf("filename required for %s", input[0])
		case ArgAddress:
			return fmt.Errorf("address required for %s", input[0])
		case ArgTarget:
			return fmt.Errorf("emulation target required for %s", input[0])
		case ArgValue:
			return fmt.Errorf("numeric argument required for %s", input[0])
		case ArgString:
			return fmt.Errorf("string argument required for %s", input[0])
		default:
			return fmt.Errorf("too few arguments for %s", input[0])
		}
	}

	return nil
}
