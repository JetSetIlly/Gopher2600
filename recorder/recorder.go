package recorder

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/peripherals"
	"gopher2600/screendigest"
	"gopher2600/television"
	"io"
	"os"
)

// Recorder records controller events to disk, intended for future playback
type Recorder struct {
	vcs    *hardware.VCS
	output *os.File
	digest *screendigest.SHA1

	headerWritten bool
}

// NewRecorder is the preferred method of implementation for the FileRecorder type
func NewRecorder(transcript string, vcs *hardware.VCS) (*Recorder, error) {
	var err error

	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return nil, errors.New(errors.RecordingError, "hardware is not suitable for recording")
	}

	rec := &Recorder{vcs: vcs}

	// create digesttv, piggybacking on the tv already being used by vcs
	rec.digest, err = screendigest.NewSHA1(vcs.TV)
	if err != nil {
		return nil, errors.New(errors.RecordingError, err)
	}

	// open file
	_, err = os.Stat(transcript)
	if os.IsNotExist(err) {
		rec.output, err = os.Create(transcript)
		if err != nil {
			return nil, errors.New(errors.RecordingError, "can't create file")
		}
	} else {
		return nil, errors.New(errors.RecordingError, "file already exists")
	}

	// delay writing of header until the first call to transcribe. we're
	// delaying this because we want to prepare the NewRecorder before we
	// attach the cartridge but writing the header requires the cartridge to
	// have been attached.
	//
	// the reason we want to create the NewRecorder before attaching the
	// cartridge is because we want to catch the setup events caused by the
	// attachement.

	return rec, nil
}

// End closes the output file.
func (rec *Recorder) End() error {
	// write the power off event to the transcript
	err := rec.Transcribe(peripherals.PanelID, peripherals.PanelPowerOff)
	if err != nil {
		return errors.New(errors.RecordingError, err)
	}

	err = rec.output.Close()
	if err != nil {
		return errors.New(errors.RecordingError, err)
	}

	return nil
}

// Transcribe implements the Transcriber interface
func (rec *Recorder) Transcribe(id peripherals.PeriphID, event peripherals.Action) error {
	var err error

	// write header if it's not been written already
	if !rec.headerWritten {
		err = rec.writeHeader()
		if err != nil {
			return errors.New(errors.RecordingError, err)
		}
		rec.headerWritten = true
	}

	// don't do anything if event is the NoEvent
	if event == peripherals.NoAction {
		return nil
	}

	// sanity checks
	if rec.output == nil {
		return errors.New(errors.RecordingError, "recording file is not open")
	}

	if rec.vcs == nil || rec.vcs.TV == nil {
		return errors.New(errors.RecordingError, "hardware is not suitable for recording")
	}

	// create line and write to file
	frame, err := rec.vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}
	scanline, err := rec.vcs.TV.GetState(television.ReqScanline)
	if err != nil {
		return err
	}
	horizpos, err := rec.vcs.TV.GetState(television.ReqHorizPos)
	if err != nil {
		return err
	}

	line := fmt.Sprintf("%v%s%v%s%v%s%v%s%v%s%v\n", id,
		fieldSep,
		event,
		fieldSep,
		frame,
		fieldSep,
		scanline,
		fieldSep,
		horizpos,
		fieldSep,
		rec.digest.String(),
	)

	n, err := io.WriteString(rec.output, line)
	if err != nil {
		return errors.New(errors.RecordingError, err)
	}
	if n != len(line) {
		return errors.New(errors.RecordingError, "output truncated")
	}

	return nil
}
