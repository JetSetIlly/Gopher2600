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

package recorder

import (
	"fmt"
	"io"
	"os"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/digest"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// Recorder transcribes user input to a file. The recorded file is intended
// for future playback. The Recorder type implements the ports.EventRecorder
// interface.
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
//
// Note that this will reset the VCS.
func NewRecorder(transcript string, vcs *hardware.VCS) (*Recorder, error) {
	var err error

	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return nil, curated.Errorf("recorder: hardware is not suitable for recording")
	}

	rec := &Recorder{
		vcs: vcs,
	}

	// we want the machine in a known state. the easiest way to do this is to
	// reset the hardware preferences
	err = vcs.Prefs.Reset()
	if err != nil {
		return nil, curated.Errorf("recorder: %v", err)
	}

	err = rec.vcs.Reset()
	if err != nil {
		return nil, curated.Errorf("recorder: %v", err)
	}

	// attach recorder to vcs peripherals, including the panel
	vcs.RIOT.Ports.AttachEventRecorder(rec)

	// video digester for playback verification
	rec.digest, err = digest.NewVideo(vcs.TV)
	if err != nil {
		return nil, curated.Errorf("recorder: %v", err)
	}

	// open file
	_, err = os.Stat(transcript)
	if os.IsNotExist(err) {
		rec.output, err = os.Create(transcript)
		if err != nil {
			return nil, curated.Errorf("recorder: can't create file")
		}
	} else {
		return nil, curated.Errorf("recorder: file already exists")
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

// End flushes all remaining events to the output file and closes it.
func (rec *Recorder) End() error {
	// write the power off event to the transcript
	err := rec.RecordEvent(ports.PanelID, ports.PanelPowerOff, nil)
	if err != nil {
		return curated.Errorf("recorder: %v", err)
	}

	err = rec.output.Close()
	if err != nil {
		return curated.Errorf("recorder: %v", err)
	}

	return nil
}

// RecordEvent implements the ports.EventRecorder interface.
func (rec *Recorder) RecordEvent(id ports.PortID, event ports.Event, value ports.EventData) error {
	var err error

	// write header if it's not been written already
	if !rec.headerWritten {
		err = rec.writeHeader()
		if err != nil {
			return curated.Errorf("recorder: %v", err)
		}
		rec.headerWritten = true
	}

	// don't do anything if event is the NoEvent
	if event == ports.NoEvent {
		return nil
	}

	// sanity checks
	if rec.output == nil {
		return curated.Errorf("recorder: recording file is not open")
	}

	if rec.vcs == nil || rec.vcs.TV == nil {
		return curated.Errorf("recorder: hardware is not suitable for recording")
	}

	// create line and write to file
	frame := rec.vcs.TV.GetState(signal.ReqFramenum)
	scanline := rec.vcs.TV.GetState(signal.ReqScanline)
	horizpos := rec.vcs.TV.GetState(signal.ReqHorizPos)

	// convert value of nil type to the empty string
	if value == nil {
		value = ""
	}

	line := fmt.Sprintf("%v%s%v%s%v%s%v%s%v%s%v%s%v\n",
		id, fieldSep,
		event, fieldSep,
		value, fieldSep,
		frame, fieldSep,
		scanline, fieldSep,
		horizpos, fieldSep,
		rec.digest.Hash(),
	)

	n, err := io.WriteString(rec.output, line)
	if err != nil {
		return curated.Errorf("recorder: %v", err)
	}
	if n != len(line) {
		return curated.Errorf("recorder: output truncated")
	}

	return nil
}
