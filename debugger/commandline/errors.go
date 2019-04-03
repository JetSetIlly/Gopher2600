package commandline

import (
	"fmt"
	"strings"
)

// ParseError is the error type for the ParseCommandTemplate function
type ParseError struct {
	definition      string
	position        int
	underlyingError error
}

func (er ParseError) Error() string {
	return fmt.Sprintf("parser error: %s", er.underlyingError)
}

// Location returns detailed information about the Error
func (er ParseError) Location() string {
	s := strings.Builder{}
	s.WriteString(er.definition)
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("%s^", strings.Repeat(" ", er.position)))
	return s.String()
}

// NewParseError is used to create a new instance of a Error
func NewParseError(defn string, position int, underlyingError error) *ParseError {
	er := new(ParseError)
	er.definition = defn
	er.position = position
	er.underlyingError = underlyingError
	return er
}
