// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package script

import (
	"gopher2600/debugger/terminal"
	"gopher2600/errors"
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
func (scr *Rescribe) TermRead(buffer []byte, _ terminal.Prompt, _ *terminal.ReadEvents) (int, error) {
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
