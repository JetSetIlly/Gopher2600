package debugger

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"os"
)

type captureScript struct {
	scriptName string
	output     *os.File
}

func (dbg *Debugger) startCaptureScript(scriptName string) (*captureScript, error) {
	cps := new(captureScript)
	cps.scriptName = scriptName

	_, err := os.Stat(scriptName)
	if os.IsNotExist(err) {
		cps.output, err = os.Create(scriptName)
		if err != nil {
			return nil, errors.NewFormattedError(errors.CaptureFileError, "can't create file")
		}
	} else {
		return nil, errors.NewFormattedError(errors.CaptureFileError, "file already exists")
	}

	return cps, nil
}

func (cps *captureScript) end() error {
	err := cps.output.Close()
	if err != nil {
		return errors.NewFormattedError(errors.CaptureFileError, err)
	}

	return nil
}

func (cps *captureScript) add(line string) error {
	line = fmt.Sprintf("%s\n", line)

	n, err := io.WriteString(cps.output, line)
	if err != nil {
		return errors.NewFormattedError(errors.CaptureFileError, err)
	}
	if n != len(line) {
		return errors.NewFormattedError(errors.CaptureFileError, "output truncated")
	}

	return nil
}
