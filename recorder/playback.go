package recorder

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/peripherals"
	"gopher2600/television"
	"gopher2600/television/renderers"
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
	CartName string
	CartHash string
	TVtype   string

	vcs       *hardware.VCS
	digest    *renderers.DigestTV
	sequences map[string]*playbackSequence
}

// NewPlayback is hte preferred method of implementation for the Playback type
func NewPlayback(transcript string, vcs *hardware.VCS) (*Playback, error) {
	var err error

	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return nil, errors.NewFormattedError(errors.PlaybackError, "no playback hardware available")
	}

	plb := &Playback{vcs: vcs}
	plb.sequences = make(map[string]*playbackSequence)

	// create digesttv, piggybacking on the tv already being used by vcs
	plb.digest, err = renderers.NewDigestTV(vcs.TV.GetSpec().ID, vcs.TV)
	if err != nil {
		return nil, errors.NewFormattedError(errors.RecordingError, err)
	}

	// open file; read the entirity of the contents; close file
	tf, err := os.Open(transcript)
	if err != nil {
		return nil, errors.NewFormattedError(errors.PlaybackError, err)
	}
	buffer, err := ioutil.ReadAll(tf)
	if err != nil {
		return nil, errors.NewFormattedError(errors.PlaybackError, err)
	}
	err = tf.Close()
	if err != nil {
		return nil, errors.NewFormattedError(errors.PlaybackError, err)
	}

	// convert file contents to an array of lines
	lines := strings.Split(string(buffer), "\n")

	// read header and perform validation checks
	err = plb.readHeader(lines)
	if err != nil {
		return nil, err
	}

	// loop through transcript and divide events according to the first field
	// (the ID)
	for i := numHeaderLines; i < len(lines)-1; i++ {
		toks := strings.Split(lines[i], fieldSep)

		// ignore lines that don't have enough fields
		if len(toks) != numFields {
			msg := fmt.Sprintf("expected %d fields at line %d", numFields, i+1)
			return nil, errors.NewFormattedError(errors.PlaybackError, msg)
		}

		// add a new playbackSequence for the id if it doesn't exist
		id := toks[fieldID]
		if _, ok := plb.sequences[id]; !ok {
			plb.sequences[id] = &playbackSequence{}
		}

		// create a new event and convert tokens accordingly
		// any errors in the transcript causes failure
		event := event{line: i + 1}

		n, err := strconv.Atoi(toks[fieldEvent])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:2], fieldSep)))
			return nil, errors.NewFormattedError(errors.PlaybackError, msg)
		}
		event.event = peripherals.Event(n)

		event.frame, err = strconv.Atoi(toks[fieldFrame])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:3], fieldSep)))
			return nil, errors.NewFormattedError(errors.PlaybackError, msg)
		}

		event.scanline, err = strconv.Atoi(toks[fieldScanline])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:4], fieldSep)))
			return nil, errors.NewFormattedError(errors.PlaybackError, msg)
		}

		event.horizpos, err = strconv.Atoi(toks[fieldHorizPos])
		if err != nil {
			msg := fmt.Sprintf("%s line %d, col %d", err, i+1, len(strings.Join(toks[:5], fieldSep)))
			return nil, errors.NewFormattedError(errors.PlaybackError, msg)
		}

		event.hash = toks[fieldHash]

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
		if nextEvent.hash != plb.digest.String() {
			msg := fmt.Sprintf("line %d", nextEvent.line)
			return peripherals.NoEvent, errors.NewFormattedError(errors.PlaybackHashError, msg)
		}

		seq.eventCt++
		return nextEvent.event, nil
	}

	// next event does not match
	return peripherals.NoEvent, nil
}
