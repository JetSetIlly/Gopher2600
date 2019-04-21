package debugger

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/gui"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type script struct {
	scriptFile string
	lines      []string
	nextLine   int
}

func (dbg *Debugger) loadScript(scriptfile string) (*script, error) {
	// open script and defer closing
	sf, err := os.Open(scriptfile)
	if err != nil {
		return nil, errors.NewFormattedError(errors.ScriptFileCannotOpen, err)
	}
	defer func() {
		_ = sf.Close()
	}()

	buffer, err := ioutil.ReadAll(sf)
	if err != nil {
		return nil, errors.NewFormattedError(errors.ScriptFileError, err)
	}

	rps := new(script)
	rps.scriptFile = scriptfile

	// convert buffer to an array of lines
	rps.lines = strings.Split(string(buffer), "\n")

	return rps, nil
}

// IsInteractive satisfies the console.UserRead interface
func (rps *script) IsInteractive() bool {
	return false
}

// UserRead implements ui.UserInput interface
func (rps *script) UserRead(buffer []byte, prompt string, _ chan gui.Event, _ func(gui.Event) error) (int, error) {
	if rps.nextLine > len(rps.lines)-1 {
		return -1, errors.NewFormattedError(errors.ScriptEnd, rps.scriptFile)
	}

	l := len(rps.lines[rps.nextLine]) + 1
	copy(buffer, []byte(rps.lines[rps.nextLine]))
	rps.nextLine++

	return l, nil
}

type scriptRecording struct {
	scriptfile string
	output     *os.File
}

func (dbg *Debugger) startScriptRecording(scriptfile string) (*scriptRecording, error) {
	rec := new(scriptRecording)
	rec.scriptfile = scriptfile

	_, err := os.Stat(scriptfile)
	if os.IsNotExist(err) {
		rec.output, err = os.Create(scriptfile)
		if err != nil {
			return nil, errors.NewFormattedError(errors.ScriptRecordingError, "can't create file")
		}
	} else {
		return nil, errors.NewFormattedError(errors.ScriptRecordingError, "file already exists")
	}

	return rec, nil
}

func (rec *scriptRecording) end() error {
	err := rec.output.Close()
	if err != nil {
		return errors.NewFormattedError(errors.ScriptRecordingError, err)
	}

	return nil
}

func (rec *scriptRecording) add(line string) error {
	line = fmt.Sprintf("%s\n", line)

	n, err := io.WriteString(rec.output, line)
	if err != nil {
		return errors.NewFormattedError(errors.ScriptRecordingError, err)
	}
	if n != len(line) {
		return errors.NewFormattedError(errors.ScriptRecordingError, "output truncated")
	}

	return nil
}
