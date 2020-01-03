package script

import (
	"gopher2600/debugger/terminal"
	"gopher2600/errors"
	"gopher2600/gui"
	"io/ioutil"
	"os"
	"strings"
)

const commentLine = "#"

// check if line is prepended with commentLine (ignoring leading spaces)
func isOutputLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), commentLine)
}

// Rescribe represents an previously scribed script. The type implements the
// terminal.UserRead interface.
type Rescribe struct {
	scriptFile string
	lines      []string
	lineCt     int
}

// RescribeScript is the preferred method of initialisation for the Rescribe
// type
func RescribeScript(scriptfile string) (*Rescribe, error) {
	// open script and defer closing
	f, err := os.Open(scriptfile)
	if err != nil {
		return nil, errors.New(errors.ScriptFileUnavailable, err)
	}
	defer func() {
		_ = f.Close()
	}()

	buffer, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.New(errors.ScriptFileError, err)
	}

	scr := &Rescribe{scriptFile: scriptfile}

	// convert buffer to an array of lines
	scr.lines = strings.Split(string(buffer), "\n")

	// pass over any lines starting with the commentLine, leaving the line
	// counter at the first input line.
	for isOutputLine(scr.lines[scr.lineCt]) {
		scr.lineCt++
		if scr.lineCt > len(scr.lines)-1 {
			// we've reached the end of the file but that's okay. subsequent
			// calls to UserRead() will result in an error, as would be
			// expected.
			return scr, nil
		}
	}

	return scr, nil
}

// IsInteractive implements the terminal.UserRead interface
func (scr *Rescribe) IsInteractive() bool {
	return false
}

// TermRead implements the terminal.UserRead interface
func (scr *Rescribe) TermRead(buffer []byte, _ terminal.Prompt, _ chan gui.Event, _ func(gui.Event) error) (int, error) {
	if scr.lineCt > len(scr.lines)-1 {
		return -1, errors.New(errors.ScriptEnd, scr.scriptFile)
	}

	command := len(scr.lines[scr.lineCt]) + 1
	copy(buffer, []byte(scr.lines[scr.lineCt]))
	scr.lineCt++

	// pass over any lines starting with the commentLine
	for scr.lineCt < len(scr.lines) && isOutputLine(scr.lines[scr.lineCt]) {
		scr.lineCt++
	}

	return command, nil
}
