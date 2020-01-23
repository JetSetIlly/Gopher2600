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

package recorder

import (
	"fmt"
	"gopher2600/digest"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/riot/input"
	"gopher2600/television"
	"io"
	"os"
)

// Recorder transcribes user input to a file. The transcribed file is intended
// for future playback. The Recorder type implements the
// riot.input.EventRecorder interface.
type Recorder struct {
	vcs    *hardware.VCS
	output *os.File

	// using video digest only to test recording validity
	digest *digest.Video

	headerWritten bool
}

// NewRecorder is the preferred method of implementation for the FileRecorder
// type. Note that attaching of the Recorder to all the ports of the VCS
// (including the panel) is implicit in this function call.
func NewRecorder(transcript string, vcs *hardware.VCS) (*Recorder, error) {
	var err error

	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return nil, errors.New(errors.RecordingError, "hardware is not suitable for recording")
	}

	rec := &Recorder{vcs: vcs}

	// attach recorder to vcs peripherals, including the panel
	vcs.HandController0.AttachEventRecorder(rec)
	vcs.HandController1.AttachEventRecorder(rec)
	vcs.Panel.AttachEventRecorder(rec)

	// video digester for playback verification
	rec.digest, err = digest.NewVideo(vcs.TV)
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

// End flushes all remaining transcription to the output file and closes it.
func (rec *Recorder) End() error {
	// write the power off event to the transcript
	err := rec.RecordEvent(input.PanelID, input.PanelPowerOff)
	if err != nil {
		return errors.New(errors.RecordingError, err)
	}

	err = rec.output.Close()
	if err != nil {
		return errors.New(errors.RecordingError, err)
	}

	return nil
}

// RecordEvent implements the input.EventRecorder interface
func (rec *Recorder) RecordEvent(id input.ID, event input.Event) error {
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
	if event == input.NoEvent {
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
		rec.digest.Hash(),
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
