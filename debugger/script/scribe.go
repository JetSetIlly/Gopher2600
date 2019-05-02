package script

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"os"
	"strings"
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
func (rec Scribe) IsActive() bool {
	return rec.file != nil
}

// StartSession a new script
func (rec *Scribe) StartSession(scriptfile string) error {
	if rec.IsActive() {
		return errors.NewFormattedError(errors.ScriptScribeError, "already active")
	}

	rec.scriptfile = scriptfile

	_, err := os.Stat(scriptfile)
	if os.IsNotExist(err) {
		rec.file, err = os.Create(scriptfile)
		if err != nil {
			return errors.NewFormattedError(errors.ScriptScribeError, "cannot create new script file")
		}
	} else {
		return errors.NewFormattedError(errors.ScriptScribeError, "file already exists")
	}

	return nil
}

// EndSession the current scribe session
func (rec *Scribe) EndSession() error {
	if !rec.IsActive() {
		return nil
	}

	defer func() {
		rec.file = nil
		rec.scriptfile = ""
		rec.playbackDepth = 0
		rec.inputLine = ""
		rec.outputLine = ""
	}()

	// make sure everything has been written to the output file
	err := rec.Commit()

	// if commit() causes an error, continue with the Close() operation and
	// return the commit() error if the close succeeds

	errClose := rec.file.Close()
	if errClose != nil {
		return errors.NewFormattedError(errors.ScriptScribeError, errClose)
	}

	return err
}

// StartPlayback indicates that a replayed script has begun
func (rec *Scribe) StartPlayback() {
	if !rec.IsActive() {
		return
	}

	rec.playbackDepth++
}

// EndPlayback indicates that a replayed script has finished
func (rec *Scribe) EndPlayback() {
	if !rec.IsActive() {
		return
	}

	rec.playbackDepth--
}

// Rollback undoes calls to WriteInput() and WriteOutput since last Commit()
func (rec *Scribe) Rollback() {
	if !rec.IsActive() {
		return
	}

	rec.inputLine = ""
	rec.outputLine = ""
}

// WriteInput writes user-input to the open script file
func (rec *Scribe) WriteInput(command string) {
	if !rec.IsActive() {
		return
	}

	rec.Commit()
	if command != "" {
		rec.inputLine = fmt.Sprintf("%s\n", command)
	}
}

// WriteOutput writes emulator-output to the open script file
func (rec *Scribe) WriteOutput(result string, args ...interface{}) {
	if !rec.IsActive() {
		return
	}

	if result == "" {
		return
	}

	result = fmt.Sprintf(result, args...)

	lines := strings.Split(result, "\n")
	for i := range lines {
		rec.outputLine = fmt.Sprintf("%s%s%s\n", rec.outputLine, outputDelimiter, lines[i])
	}
}

// Commit most recent calls to WriteInput() and WriteOutput()
func (rec *Scribe) Commit() error {
	if !rec.IsActive() {
		return nil
	}

	defer func() {
		rec.inputLine = ""
		rec.outputLine = ""
	}()

	if rec.playbackDepth > 0 {
		return nil
	}

	if rec.inputLine != "" {
		n, err := io.WriteString(rec.file, rec.inputLine)
		if err != nil {
			return errors.NewFormattedError(errors.ScriptScribeError, err)
		}
		if n != len(rec.inputLine) {
			return errors.NewFormattedError(errors.ScriptScribeError, "output truncated")
		}
	}

	if rec.outputLine != "" {
		n, err := io.WriteString(rec.file, rec.outputLine)
		if err != nil {
			return errors.NewFormattedError(errors.ScriptScribeError, err)
		}
		if n != len(rec.outputLine) {
			return errors.NewFormattedError(errors.ScriptScribeError, "output truncated")
		}
	}

	return nil
}
