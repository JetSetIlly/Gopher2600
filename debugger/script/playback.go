package script

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/gui"
	"io/ioutil"
	"os"
	"strings"
)

const outputDelimiter = ">> "

// Playback represents an existing script that has been loaded for playback
type Playback struct {
	scriptFile    string
	lines         []string
	nextLine      int
	expectedOuput string
}

// StartPlayback is the preferred method of initialisation for a script
func StartPlayback(scriptfile string) (*Playback, error) {
	// open script and defer closing
	sf, err := os.Open(scriptfile)
	if err != nil {
		return nil, errors.NewFormattedError(errors.ScriptFileUnavailable, err)
	}
	defer func() {
		_ = sf.Close()
	}()

	buffer, err := ioutil.ReadAll(sf)
	if err != nil {
		return nil, errors.NewFormattedError(errors.ScriptFileError, err)
	}

	rps := &Playback{scriptFile: scriptfile}

	// convert buffer to an array of lines
	rps.lines = strings.Split(string(buffer), "\n")

	// pass over any leading "output" lines. this shouldn't happen unless the
	// script has been manually edited
	for strings.HasPrefix(rps.lines[rps.nextLine], outputDelimiter) {
		rps.nextLine++
		if rps.nextLine > len(rps.lines)-1 {
			// we've reached the end of the file but that's okay. subsequent
			// calls to UserRead() will result in an error, as would be
			// expected.
			return rps, nil
		}
	}

	return rps, nil
}

// IsInteractive implements the console.UserRead interface
func (rps *Playback) IsInteractive() bool {
	return false
}

// UserRead implements ui.UserInput interface
func (rps *Playback) UserRead(buffer []byte, prompt string, _ chan gui.Event, _ func(gui.Event) error) (int, error) {
	if rps.nextLine > len(rps.lines)-1 {
		return -1, errors.NewFormattedError(errors.ScriptEnd, rps.scriptFile)
	}

	command := len(rps.lines[rps.nextLine]) + 1
	copy(buffer, []byte(rps.lines[rps.nextLine]))
	rps.nextLine++

	// build output expected as a result of running the command
	rps.expectedOuput = ""
	for rps.nextLine < len(rps.lines) && strings.HasPrefix(rps.lines[rps.nextLine], outputDelimiter) {
		rps.expectedOuput = fmt.Sprintf("%s%s", rps.expectedOuput, rps.lines[rps.nextLine])
		rps.nextLine++
	}

	return command, nil
}
