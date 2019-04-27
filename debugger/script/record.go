package script

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"os"
	"strings"
)

// Recorder can be used again after a start()/end() cycle. isRecording()
// can be used to detect if a recording is currently taking place but it is
// safe not to do because most functions silently fail if there is no current
// recording.
type Recorder struct {
	file       *os.File
	scriptfile string

	// the depth of script openings during the recording of a new script
	playbackDepth int

	inputLine  string
	outputLine string
}

// IsRecording returns true if a recording is currently active
func (rec Recorder) IsRecording() bool {
	return rec.file != nil
}

// Start a new recording
// can be used without explicit IsRecording() check
func (rec *Recorder) Start(scriptfile string) error {
	if rec.IsRecording() {
		return errors.NewFormattedError(errors.ScriptRecordingError, "recording already active")
	}

	rec.scriptfile = scriptfile

	_, err := os.Stat(scriptfile)
	if os.IsNotExist(err) {
		rec.file, err = os.Create(scriptfile)
		if err != nil {
			return errors.NewFormattedError(errors.ScriptRecordingError, "can't create file")
		}
	} else {
		return errors.NewFormattedError(errors.ScriptRecordingError, "file already exists")
	}

	return nil
}

// End the current recording
// can be used without explicit IsRecording() check
func (rec *Recorder) End() error {
	if !rec.IsRecording() {
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
		return errors.NewFormattedError(errors.ScriptRecordingError, errClose)
	}

	return err
}

// StartPlayback indicates that a replayed script has begun
// can be used without explicit IsRecording() check
func (rec *Recorder) StartPlayback() {
	if !rec.IsRecording() {
		return
	}

	rec.playbackDepth++
}

// EndPlayback indicates that a replayed script has finished
// can be used without explicit IsRecording() check
func (rec *Recorder) EndPlayback() {
	if !rec.IsRecording() {
		return
	}

	rec.playbackDepth--
}

// Rollback undoes calls to WriteInput() and WriteOutput since last Commit()
// can be used without explicit IsRecording() check
func (rec *Recorder) Rollback() {
	if !rec.IsRecording() {
		return
	}

	rec.inputLine = ""
	rec.outputLine = ""
}

// WriteInput puts debugger input to open recording file
// can be used without explicit IsRecording() check
func (rec *Recorder) WriteInput(command string) {
	if !rec.IsRecording() {
		return
	}

	rec.Commit()
	if command != "" {
		rec.inputLine = fmt.Sprintf("%s\n", command)
	}
}

// WriteOutput puts debugger output to open recording file
// can be used without explicit IsRecording() check
func (rec *Recorder) WriteOutput(result string, args ...interface{}) {
	if !rec.IsRecording() {
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

// Commit command and result to the output file
// can be used without explicit IsRecording() check
func (rec *Recorder) Commit() error {
	if !rec.IsRecording() {
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
			return errors.NewFormattedError(errors.ScriptRecordingError, err)
		}
		if n != len(rec.inputLine) {
			return errors.NewFormattedError(errors.ScriptRecordingError, "output truncated")
		}
	}

	if rec.outputLine != "" {
		n, err := io.WriteString(rec.file, rec.outputLine)
		if err != nil {
			return errors.NewFormattedError(errors.ScriptRecordingError, err)
		}
		if n != len(rec.outputLine) {
			return errors.NewFormattedError(errors.ScriptRecordingError, "output truncated")
		}
	}

	return nil
}
