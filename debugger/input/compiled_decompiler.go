package input

import (
	"fmt"
	"strings"
)

// string functions for the three types:
//
//  o Commands
//  o commandArgList
//  o commandArg
//
// calling String() on Commands should reproduce the template from which the
// commands were compiled

func (cmd Commands) String() string {
	s := strings.Builder{}
	for k, v := range cmd {
		s.WriteString(k)
		s.WriteString(fmt.Sprintf("%s", v))
		s.WriteString("\n")
	}
	return s.String()
}

func (a commandArgList) String() string {
	s := strings.Builder{}
	for i := range a {
		s.WriteString(fmt.Sprintf(" %s", a[i]))
	}
	return s.String()
}

func (c commandArg) String() string {
	switch c.typ {
	case argKeyword:
		s := "["
		switch values := c.values.(type) {
		case []string:
			for i := range values {
				s = fmt.Sprintf("%s%s|", s, values[i])
			}
			s = strings.TrimSuffix(s, "|")
		case *Commands:
			s = fmt.Sprintf("%s<commands>", s)
		default:
			s = fmt.Sprintf("%s%T", s, values)
		}
		return fmt.Sprintf("%s]", s)
	case argFile:
		return "%F"
	case argValue:
		return "%V"
	case argString:
		return "%S"
	case argIndeterminate:
		return "%*"
	}
	return "!!"
}
