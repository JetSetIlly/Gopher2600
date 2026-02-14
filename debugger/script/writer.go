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
)

// Write can be used again after a start()/end() cycle. isWriting()
// can be used to detect if a script is currently being captured but it is
// safe not to do because most functions silently fail if there is no current
// active session.
type Write struct {
	file       *os.File
	scriptfile string

	// the depth of script openings during the writing of a new script
	playbackDepth int

	inputLine  string
	outputLine string
}

// IsActive returns true if a script is currently being capture.
func (w Write) IsActive() bool {
	return w.file != nil
}

// StartSession a new script.
func (w *Write) StartSession(scriptfile string) error {
	if w.IsActive() {
		return fmt.Errorf("script: already active")
	}

	w.scriptfile = scriptfile

	_, err := os.Stat(scriptfile)
	if os.IsNotExist(err) {
		w.file, err = os.Create(scriptfile)
		if err != nil {
			return fmt.Errorf("script: cannot create file (%s)", scriptfile)
		}
	} else {
		return fmt.Errorf("script: file already exists (%s)", scriptfile)
	}

	return nil
}

// EndSession the current write session.
func (w *Write) EndSession() (rerr error) {
	if !w.IsActive() {
		return nil
	}

	defer func() {
		w.file = nil
		w.scriptfile = ""
		w.playbackDepth = 0
		w.inputLine = ""
		w.outputLine = ""
	}()

	defer func() {
		err := w.file.Close()
		if err != nil {
			rerr = fmt.Errorf("script: %w", err)
		}
	}()

	// make sure everything has been written to the output file
	return w.Commit()
}

// StartPlayback indicates that a replayed script has begun.
func (w *Write) StartPlayback() error {
	if !w.IsActive() {
		return nil
	}

	err := w.Commit()
	if err != nil {
		return err
	}

	w.playbackDepth++

	return nil
}

// EndPlayback indicates that a replayed script has finished.
func (w *Write) EndPlayback() error {
	if !w.IsActive() {
		return nil
	}

	err := w.Commit()
	if err != nil {
		return err
	}

	w.playbackDepth--

	return nil
}

// Rollback undoes calls to WriteInput() and WriteOutput since last Commit().
func (w *Write) Rollback() {
	if !w.IsActive() {
		return
	}

	w.inputLine = ""
	w.outputLine = ""
}

// WriteInput writes user-input to the open script file.
func (w *Write) WriteInput(command string) error {
	if !w.IsActive() || w.playbackDepth > 0 {
		return nil
	}

	err := w.Commit()
	if err != nil {
		return err
	}

	if command != "" {
		w.inputLine = fmt.Sprintf("%s\n", command)
	}

	return nil
}

// Commit most recent calls to WriteInput() and WriteOutput().
func (w *Write) Commit() error {
	if !w.IsActive() {
		return nil
	}

	defer func() {
		w.inputLine = ""
		w.outputLine = ""
	}()

	if w.inputLine != "" {
		n, err := io.WriteString(w.file, w.inputLine)
		if err != nil {
			return fmt.Errorf("script: %w", err)
		}
		if n != len(w.inputLine) {
			return fmt.Errorf("script: output truncated")
		}
	}

	if w.outputLine != "" {
		n, err := io.WriteString(w.file, w.outputLine)
		if err != nil {
			return fmt.Errorf("script: %w", err)
		}
		if n != len(w.outputLine) {
			return fmt.Errorf("script: output truncated")
		}
	}

	return nil
}
