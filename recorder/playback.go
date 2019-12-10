package recorder

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/digest"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/peripherals"
	"gopher2600/television"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type event struct {
	event    peripherals.Action
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

// Playback is an implementation of the controller interface. it reads from an
// existing recording file and responds to GetInput() requests
type Playback struct {
	transcript string

	CartLoad cartridgeloader.Loader
	TVtype   string

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
// playback
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

// NewPlayback is the preferred method of implementation for the Playback type
func NewPlayback(transcript string) (*Playback, error) {
	var err error

	plb := &Playback{transcript: transcript}
	plb.sequences = make([]*playbackSequence, peripherals.NumPeriphIDs)
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
		id := peripherals.PeriphID(n)

		// create a new event and convert tokens accordingly
		// any errors in the transcript causes failure
		event := event{line: i + 1}

		n, err = strconv.Atoi(toks[fieldEvent])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldEvent+1], fieldSep)))
			return nil, errors.New(errors.PlaybackError, msg)
		}
		event.event = peripherals.Action(n)

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
// controller interface) to the supplied VCS
func (plb *Playback) AttachToVCS(vcs *hardware.VCS) error {
	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return errors.New(errors.PlaybackError, "no playback hardware available")
	}
	plb.vcs = vcs

	// validate header
	if plb.vcs.TV.GetSpec().ID != plb.TVtype {
		return errors.New(errors.PlaybackError, "current TV type does not match that in the recording")
	}

	var err error

	plb.digest, err = digest.NewVideo(plb.vcs.TV)
	if err != nil {
		return errors.New(errors.RecordingError, err)
	}

	// attach playback to controllers
	vcs.Ports.Player0.Attach(plb)
	vcs.Ports.Player1.Attach(plb)
	vcs.Panel.Attach(plb)

	return nil
}

// GetInput implements peripherals.Controller interface
func (plb *Playback) GetInput(id peripherals.PeriphID) (peripherals.Action, error) {
	// there's no events for this id at all
	seq := plb.sequences[id]

	// we've reached the end of the list of events for this id
	if seq.eventCt >= len(seq.events) {
		return peripherals.NoAction, nil
	}

	// get current state of the television
	frame, err := plb.vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return peripherals.NoAction, errors.New(errors.PlaybackError, err)
	}
	scanline, err := plb.vcs.TV.GetState(television.ReqScanline)
	if err != nil {
		return peripherals.NoAction, errors.New(errors.PlaybackError, err)
	}
	horizpos, err := plb.vcs.TV.GetState(television.ReqHorizPos)
	if err != nil {
		return peripherals.NoAction, errors.New(errors.PlaybackError, err)
	}

	// compare current state with the recording
	nextEvent := seq.events[seq.eventCt]
	if frame == nextEvent.frame && scanline == nextEvent.scanline && horizpos == nextEvent.horizpos {
		if nextEvent.hash != plb.digest.Hash() {
			return peripherals.NoAction, errors.New(errors.PlaybackHashError, fmt.Sprintf("line %d", nextEvent.line))
		}

		seq.eventCt++
		return nextEvent.event, nil
	}

	// next event does not match
	return peripherals.NoAction, nil
}
