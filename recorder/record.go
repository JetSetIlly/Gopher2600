package recorder

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/peripherals"
	"gopher2600/television"
	"io"
	"os"
)

const fieldSep = ", "
const numFields = 5

// Recorder records controller events to disk, intended for future playback
type Recorder struct {
	vcs    *hardware.VCS
	output *os.File
}

// NewRecorder is the preferred method of implementation for the FileRecorder type
func NewRecorder(transcript string, vcs *hardware.VCS) (*Recorder, error) {
	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return nil, errors.NewFormattedError(errors.RecordingError, "hardware is not suitable for recording")
	}

	scr := &Recorder{vcs: vcs}

	// open file
	_, err := os.Stat(transcript)
	if os.IsNotExist(err) {
		scr.output, err = os.Create(transcript)
		if err != nil {
			return nil, errors.NewFormattedError(errors.RecordingError, "can't create file")
		}
	} else {
		return nil, errors.NewFormattedError(errors.RecordingError, "file already exists")
	}

	// add header information
	tvspec, err := scr.vcs.TV.GetState(television.ReqTVSpec)
	if err != nil {
		scr.output.Close()
		return nil, errors.NewFormattedError(errors.RecordingError, err)
	}

	line := fmt.Sprintf("%v\n", tvspec)

	n, err := io.WriteString(scr.output, line)
	if err != nil {
		scr.output.Close()
		return nil, errors.NewFormattedError(errors.RecordingError, err)
	}
	if n != len(line) {
		scr.output.Close()
		return nil, errors.NewFormattedError(errors.RecordingError, "output truncated")
	}

	return scr, nil
}

// End closes the output file.
func (scr *Recorder) End() error {
	err := scr.output.Close()
	if err != nil {
		return errors.NewFormattedError(errors.RecordingError, err)
	}

	return nil
}

// Transcribe implements the Transcriber interface
func (scr *Recorder) Transcribe(id string, event peripherals.Event) error {
	// don't do anything if event is the NoEvent
	if event == peripherals.NoEvent {
		return nil
	}

	// sanity checks
	if scr.output == nil {
		return errors.NewFormattedError(errors.RecordingError, "recording file is not open")
	}

	if scr.vcs == nil || scr.vcs.TV == nil {
		return errors.NewFormattedError(errors.RecordingError, "hardware is not suitable for recording")
	}

	// create line and write to file
	frame, err := scr.vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}
	scanline, err := scr.vcs.TV.GetState(television.ReqScanline)
	if err != nil {
		return err
	}
	horizpos, err := scr.vcs.TV.GetState(television.ReqHorizPos)
	if err != nil {
		return err
	}

	line := fmt.Sprintf("%v%s%v%s%v%s%v%s%v\n", id, fieldSep, event, fieldSep, frame, fieldSep, scanline, fieldSep, horizpos)

	n, err := io.WriteString(scr.output, line)
	if err != nil {
		return errors.NewFormattedError(errors.RecordingError, err)
	}
	if n != len(line) {
		return errors.NewFormattedError(errors.RecordingError, "output truncated")
	}

	return nil
}
