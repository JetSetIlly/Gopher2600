package recorder

import (
	"fmt"
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
	event    peripherals.Event
	frame    int
	scanline int
	horizpos int
}

type playbackSequence struct {
	events  []event
	eventCt int
}

// Playback is an implementation of the controller interface. it reads from an
// existing recording file and responds to GetInput() requests
type Playback struct {
	vcs       *hardware.VCS
	sequences map[string]*playbackSequence
}

// NewPlayback is hte preferred method of implementation for the Playback type
func NewPlayback(transcript string, vcs *hardware.VCS) (*Playback, error) {
	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return nil, errors.NewFormattedError(errors.PlaybackError, "no playback hardware available")
	}

	plb := &Playback{vcs: vcs}
	plb.sequences = make(map[string]*playbackSequence)

	// open file
	tf, err := os.Open(transcript)
	if err != nil {
		return nil, errors.NewFormattedError(errors.PlaybackError, err)
	}

	buffer, err := ioutil.ReadAll(tf)
	if err != nil {
		return nil, errors.NewFormattedError(errors.PlaybackError, err)
	}

	_ = tf.Close()

	// convert buffer to an array of lines
	lines := strings.Split(string(buffer), "\n")

	// read header
	tvspec := plb.vcs.TV.GetSpec()
	if tvspec.ID != lines[0] {
		return nil, errors.NewFormattedError(errors.PlaybackError, "current TV type does not match that in the recording")
	}

	// loop through transcript and divide events according to the first field
	// (the ID)
	for i := 1; i < len(lines)-1; i++ {
		toks := strings.Split(lines[i], fieldSep)

		// ignore lines that don't have enough fields
		if len(toks) != numFields {
			continue
		}

		// add a new playbackSequence for the id if it doesn't exist
		id := toks[0]
		if _, ok := plb.sequences[id]; !ok {
			plb.sequences[id] = &playbackSequence{}
		}

		// create a new event and convert tokens accordingly
		// any errors in the transcript causes failure
		event := event{}

		n, err := strconv.Atoi(toks[1])
		if err != nil {
			msg := fmt.Sprintf("line %d, col %d", i+1, len(strings.Join(toks[:2], fieldSep)))
			return nil, errors.NewFormattedError(errors.PlaybackError, msg)
		}
		event.event = peripherals.Event(n)

		event.frame, err = strconv.Atoi(toks[2])
		if err != nil {
			msg := fmt.Sprintf("line %d, col %d", i+1, len(strings.Join(toks[:3], fieldSep)))
			return nil, errors.NewFormattedError(errors.PlaybackError, msg)
		}

		event.scanline, err = strconv.Atoi(toks[3])
		if err != nil {
			msg := fmt.Sprintf("line %d, col %d", i+1, len(strings.Join(toks[:4], fieldSep)))
			return nil, errors.NewFormattedError(errors.PlaybackError, msg)
		}

		event.horizpos, err = strconv.Atoi(toks[4])
		if err != nil {
			msg := fmt.Sprintf("line %d, col %d", i+1, len(strings.Join(toks[:5], fieldSep)))
			return nil, errors.NewFormattedError(errors.PlaybackError, msg)
		}

		// add new event to list of events in the correct playback sequence
		seq := plb.sequences[id]
		seq.events = append(seq.events, event)
	}

	return plb, nil
}

// GetInput implements peripherals.Controller interface
func (plb *Playback) GetInput(id string) (peripherals.Event, error) {
	// there's no events for this id at all
	seq, ok := plb.sequences[id]
	if !ok {
		return peripherals.NoEvent, nil
	}

	// we've reached the end of the list of events for this id
	if seq.eventCt >= len(seq.events) {
		return peripherals.NoEvent, nil
	}

	// get current state of the television
	frame, err := plb.vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return peripherals.NoEvent, errors.NewFormattedError(errors.PlaybackError, err)
	}
	scanline, err := plb.vcs.TV.GetState(television.ReqScanline)
	if err != nil {
		return peripherals.NoEvent, errors.NewFormattedError(errors.PlaybackError, err)
	}
	horizpos, err := plb.vcs.TV.GetState(television.ReqHorizPos)
	if err != nil {
		return peripherals.NoEvent, errors.NewFormattedError(errors.PlaybackError, err)
	}

	// compare current state with the state in the transcript
	nextEvent := seq.events[seq.eventCt]
	if frame == nextEvent.frame && scanline == nextEvent.scanline && horizpos == nextEvent.horizpos {
		seq.eventCt++
		return nextEvent.event, nil
	}

	// next event does not match
	return peripherals.NoEvent, nil
}
