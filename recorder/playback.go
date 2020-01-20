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
	"gopher2600/cartridgeloader"
	"gopher2600/digest"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/riot/input"
	"gopher2600/television"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type event struct {
	event    input.Event
	frame    int
	scanline int
	horizpos int
	hash     string

	// the line in the recording file the playback event appears
	line int
}

type playbackSequence struct {
	events  []event
	eventCt int
}

// Playback is used to reperform the user input recorded in a previously transcribed
// file. It implements the input.Controller interface.
type Playback struct {
	transcript string

	CartLoad cartridgeloader.Loader
	TVSpec   string

	sequences []*playbackSequence
	vcs       *hardware.VCS
	digest    *digest.Video

	// the last frame where an event occurs
	endFrame int
}

func (plb Playback) String() string {
	currFrame, err := plb.digest.GetState(television.ReqFramenum)
	if err != nil {
		currFrame = plb.endFrame
	}
	return fmt.Sprintf("%d/%d (%.1f%%)", currFrame, plb.endFrame, 100*(float64(currFrame)/float64(plb.endFrame)))
}

// EndFrame returns true if emulation has gone past the last frame of the
// playback.
func (plb Playback) EndFrame() (bool, error) {
	currFrame, err := plb.digest.GetState(television.ReqFramenum)
	if err != nil {
		return false, errors.New(errors.RegressionPlaybackError, err)
	}

	if currFrame > plb.endFrame {
		return true, nil
	}

	return false, nil

}

// NewPlayback is the preferred method of implementation for the Playback type.
func NewPlayback(transcript string) (*Playback, error) {
	var err error

	plb := &Playback{transcript: transcript}
	plb.sequences = make([]*playbackSequence, input.NumIDs)
	for i := range plb.sequences {
		plb.sequences[i] = &playbackSequence{}
	}

	tf, err := os.Open(transcript)
	if err != nil {
		return nil, errors.New(errors.PlaybackError, err)
	}
	buffer, err := ioutil.ReadAll(tf)
	if err != nil {
		return nil, errors.New(errors.PlaybackError, err)
	}
	err = tf.Close()
	if err != nil {
		return nil, errors.New(errors.PlaybackError, err)
	}

	// convert file contents to an array of lines
	lines := strings.Split(string(buffer), "\n")

	// read header and perform validation checks
	err = plb.readHeader(lines)
	if err != nil {
		return nil, err
	}

	// loop through transcript and divide events according to the first field
	// (the peripheral ID)
	for i := numHeaderLines; i < len(lines)-1; i++ {
		toks := strings.Split(lines[i], fieldSep)

		// ignore lines that don't have enough fields
		if len(toks) != numFields {
			msg := fmt.Sprintf("expected %d fields at line %d", numFields, i+1)
			return nil, errors.New(errors.PlaybackError, msg)
		}

		// add a new playbackSequence for the id if it doesn't exist
		n, err := strconv.Atoi(toks[fieldID])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldID+1], fieldSep)))
			return nil, errors.New(errors.PlaybackError, msg)
		}
		id := input.ID(n)

		// create a new event and convert tokens accordingly
		// any errors in the transcript causes failure
		event := event{line: i + 1}

		n, err = strconv.Atoi(toks[fieldEvent])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldEvent+1], fieldSep)))
			return nil, errors.New(errors.PlaybackError, msg)
		}
		event.event = input.Event(n)

		event.frame, err = strconv.Atoi(toks[fieldFrame])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldFrame+1], fieldSep)))
			return nil, errors.New(errors.PlaybackError, msg)
		}

		// assuming that frames are listed in order in the file. update
		// endFrame with the most recent frame every time
		plb.endFrame = event.frame

		event.scanline, err = strconv.Atoi(toks[fieldScanline])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldScanline+1], fieldSep)))
			return nil, errors.New(errors.PlaybackError, msg)
		}

		event.horizpos, err = strconv.Atoi(toks[fieldHorizPos])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldHorizPos+1], fieldSep)))
			return nil, errors.New(errors.PlaybackError, msg)
		}

		event.hash = toks[fieldHash]

		// add new event to list of events in the correct playback sequence
		seq := plb.sequences[id]
		seq.events = append(seq.events, event)
	}

	return plb, nil
}

// AttachToVCS attaches the playback instance (an implementation of the
// controller interface) to all the ports of the VCS, including the panel.
func (plb *Playback) AttachToVCS(vcs *hardware.VCS) error {
	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return errors.New(errors.PlaybackError, "no playback hardware available")
	}
	plb.vcs = vcs

	// validate header. keep it simple and disallow any difference in tv
	// specification. some combinations may work but there's no compelling
	// reason to figure that out just now.
	if plb.vcs.TV.SpecIDOnCreation() != plb.TVSpec {
		return errors.New(errors.PlaybackError,
			fmt.Sprintf("recording was made with the %s TV spec. trying to playback with a TV spec of %s.",
				plb.TVSpec, vcs.TV.SpecIDOnCreation()))
	}

	var err error

	plb.digest, err = digest.NewVideo(plb.vcs.TV)
	if err != nil {
		return errors.New(errors.RecordingError, err)
	}

	// attach playback to controllers
	vcs.Player0.Attach(plb)
	vcs.Player1.Attach(plb)
	vcs.Panel.Attach(plb)

	return nil
}

// CheckInput implements the input.Controller interface.
func (plb *Playback) CheckInput(id input.ID) (input.Event, error) {
	// there's no events for this id at all
	seq := plb.sequences[id]

	// we've reached the end of the list of events for this id
	if seq.eventCt >= len(seq.events) {
		return input.NoEvent, nil
	}

	// get current state of the television
	frame, err := plb.vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return input.NoEvent, errors.New(errors.PlaybackError, err)
	}
	scanline, err := plb.vcs.TV.GetState(television.ReqScanline)
	if err != nil {
		return input.NoEvent, errors.New(errors.PlaybackError, err)
	}
	horizpos, err := plb.vcs.TV.GetState(television.ReqHorizPos)
	if err != nil {
		return input.NoEvent, errors.New(errors.PlaybackError, err)
	}

	// compare current state with the recording
	nextEvent := seq.events[seq.eventCt]
	if frame == nextEvent.frame && scanline == nextEvent.scanline && horizpos == nextEvent.horizpos {
		if nextEvent.hash != plb.digest.Hash() {
			return input.NoEvent, errors.New(errors.PlaybackHashError, fmt.Sprintf("line %d", nextEvent.line))
		}

		seq.eventCt++
		return nextEvent.event, nil
	}

	// next event does not match
	return input.NoEvent, nil
}
