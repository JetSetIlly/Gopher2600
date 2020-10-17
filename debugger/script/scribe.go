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

package script

import (
	"fmt"
	"io"
	"os"

	"github.com/jetsetilly/gopher2600/curated"
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

// IsActive returns true if a script is currently being capture.
func (scr Scribe) IsActive() bool {
	return scr.file != nil
}

// StartSession a new script.
func (scr *Scribe) StartSession(scriptfile string) error {
	if scr.IsActive() {
		return curated.Errorf("script scribe already active")
	}

	scr.scriptfile = scriptfile

	_, err := os.Stat(scriptfile)
	if os.IsNotExist(err) {
		scr.file, err = os.Create(scriptfile)
		if err != nil {
			return curated.Errorf("cannot create new script file")
		}
	} else {
		return curated.Errorf("file already exists")
	}

	return nil
}

// EndSession the current scribe session.
func (scr *Scribe) EndSession() (rerr error) {
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

	defer func() {
		err := scr.file.Close()
		if err != nil {
			rerr = curated.Errorf("script: scripe: %v", err)
		}
	}()

	// make sure everything has been written to the output file
	return scr.Commit()
}

// StartPlayback indicates that a replayed script has begun.
func (scr *Scribe) StartPlayback() error {
	if !scr.IsActive() {
		return nil
	}

	err := scr.Commit()
	if err != nil {
		return err
	}

	scr.playbackDepth++

	return nil
}

// EndPlayback indicates that a replayed script has finished.
func (scr *Scribe) EndPlayback() error {
	if !scr.IsActive() {
		return nil
	}

	err := scr.Commit()
	if err != nil {
		return err
	}

	scr.playbackDepth--

	return nil
}

// Rollback undoes calls to WriteInput() and WriteOutput since last Commit().
func (scr *Scribe) Rollback() {
	if !scr.IsActive() {
		return
	}

	scr.inputLine = ""
	scr.outputLine = ""
}

// WriteInput writes user-input to the open script file.
func (scr *Scribe) WriteInput(command string) error {
	if !scr.IsActive() || scr.playbackDepth > 0 {
		return nil
	}

	err := scr.Commit()
	if err != nil {
		return err
	}

	if command != "" {
		scr.inputLine = fmt.Sprintf("%s\n", command)
	}

	return nil
}

// Commit most recent calls to WriteInput() and WriteOutput().
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
			return curated.Errorf("script: scribe: %v", err)
		}
		if n != len(scr.inputLine) {
			return curated.Errorf("script: scribe output truncated")
		}
	}

	if scr.outputLine != "" {
		n, err := io.WriteString(scr.file, scr.outputLine)
		if err != nil {
			return curated.Errorf("script: scribe: %v", err)
		}
		if n != len(scr.outputLine) {
			return curated.Errorf("script: scribe output truncated")
		}
	}

	return nil
}
