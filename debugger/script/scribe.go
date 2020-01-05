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
	"fmt"
	"gopher2600/errors"
	"io"
	"os"
)

// Scribe can be used again after a start()/end() cycle. isWriting()
// can be used to detect if a script is currently being captured but it is
// safe not to do because most functions silently fail if there is no current
// active session.
type Scribe struct {
	file       *os.File
	scriptfile string

	// the depth of script openings during the writing of a new script
	playbackDepth int

	inputLine  string
	outputLine string
}

// IsActive returns true if a script is currently being capture
func (scr Scribe) IsActive() bool {
	return scr.file != nil
}

// StartSession a new script
func (scr *Scribe) StartSession(scriptfile string) error {
	if scr.IsActive() {
		return errors.New(errors.ScriptScribeError, "already active")
	}

	scr.scriptfile = scriptfile

	_, err := os.Stat(scriptfile)
	if os.IsNotExist(err) {
		scr.file, err = os.Create(scriptfile)
		if err != nil {
			return errors.New(errors.ScriptScribeError, "cannot create new script file")
		}
	} else {
		return errors.New(errors.ScriptScribeError, "file already exists")
	}

	return nil
}

// EndSession the current scribe session
func (scr *Scribe) EndSession() error {
	if !scr.IsActive() {
		return nil
	}

	defer func() {
		scr.file = nil
		scr.scriptfile = ""
		scr.playbackDepth = 0
		scr.inputLine = ""
		scr.outputLine = ""
	}()

	// make sure everything has been written to the output file
	err := scr.Commit()

	// if commit() causes an error, continue with the Close() operation and
	// return the commit() error if the close succeeds

	errClose := scr.file.Close()
	if errClose != nil {
		return errors.New(errors.ScriptScribeError, errClose)
	}

	return err
}

// StartPlayback indicates that a replayed script has begun
func (scr *Scribe) StartPlayback() {
	if !scr.IsActive() {
		return
	}
	scr.Commit()
	scr.playbackDepth++
}

// EndPlayback indicates that a replayed script has finished
func (scr *Scribe) EndPlayback() {
	if !scr.IsActive() {
		return
	}
	scr.Commit()
	scr.playbackDepth--
}

// Rollback undoes calls to WriteInput() and WriteOutput since last Commit()
func (scr *Scribe) Rollback() {
	if !scr.IsActive() {
		return
	}

	scr.inputLine = ""
	scr.outputLine = ""
}

// WriteInput writes user-input to the open script file
func (scr *Scribe) WriteInput(command string) {
	if !scr.IsActive() || scr.playbackDepth > 0 {
		return
	}

	scr.Commit()
	if command != "" {
		scr.inputLine = fmt.Sprintf("%s\n", command)
	}
}

// Commit most scrent calls to WriteInput() and WriteOutput()
func (scr *Scribe) Commit() error {
	if !scr.IsActive() {
		return nil
	}

	defer func() {
		scr.inputLine = ""
		scr.outputLine = ""
	}()

	if scr.inputLine != "" {
		n, err := io.WriteString(scr.file, scr.inputLine)
		if err != nil {
			return errors.New(errors.ScriptScribeError, err)
		}
		if n != len(scr.inputLine) {
			return errors.New(errors.ScriptScribeError, "output truncated")
		}
	}

	if scr.outputLine != "" {
		n, err := io.WriteString(scr.file, scr.outputLine)
		if err != nil {
			return errors.New(errors.ScriptScribeError, err)
		}
		if n != len(scr.outputLine) {
			return errors.New(errors.ScriptScribeError, "output truncated")
		}
	}

	return nil
}
