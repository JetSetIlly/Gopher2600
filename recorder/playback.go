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
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/digest"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

type playbackEntry struct {
	portID   ports.PortID
	event    ports.Event
	value    ports.EventData
	frame    int
	scanline int
	horizpos int
	hash     string

	// the line in the recording file the playback event appears
	line int
}

// Playback is used to reperform the user input recorded in a previously
// recorded file. It implements the ports.Playback interface.
type Playback struct {
	transcript string

	CartLoad cartridgeloader.Loader
	TVSpec   string

	sequence []playbackEntry
	seqCt    int

	vcs    *hardware.VCS
	digest *digest.Video

	// the last frame where an event occurs
	endFrame int
}

func (plb Playback) String() string {
	currFrame := plb.digest.GetState(signal.ReqFramenum)
	return fmt.Sprintf("%d/%d (%.1f%%)", currFrame, plb.endFrame, 100*(float64(currFrame)/float64(plb.endFrame)))
}

// EndFrame returns true if emulation has gone past the last frame of the
// playback.
func (plb Playback) EndFrame() (bool, error) {
	currFrame := plb.digest.GetState(signal.ReqFramenum)
	if currFrame > plb.endFrame {
		return true, nil
	}

	return false, nil
}

// NewPlayback is the preferred method of implementation for the Playback type.
func NewPlayback(transcript string) (*Playback, error) {
	var err error

	plb := &Playback{
		transcript: transcript,
		sequence:   make([]playbackEntry, 0),
	}

	tf, err := os.Open(transcript)
	if err != nil {
		return nil, curated.Errorf("playback: %v", err)
	}
	buffer, err := ioutil.ReadAll(tf)
	if err != nil {
		return nil, curated.Errorf("playback: %v", err)
	}
	err = tf.Close()
	if err != nil {
		return nil, curated.Errorf("playback: %v", err)
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
			return nil, curated.Errorf("playback: expected %d fields at line %d", numFields, i+1)
		}

		// add a new playbackSequence for the id if it doesn't exist
		n, err := strconv.Atoi(toks[fieldID])
		if err != nil {
			return nil, curated.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldID+1], fieldSep)))
		}

		// create a new entry and convert tokens accordingly
		// any errors in the transcript causes failure
		entry := playbackEntry{
			portID: ports.PortID(n),
			line:   i + 1,
		}

		// no need to convert event field
		entry.event = ports.Event(toks[fieldEvent])

		// parse entry value into the correct type
		entry.value = parseEventData(toks[fieldEventData])

		// special condition for KeyboardDown and KeyboardUp events.
		//
		// we don't like special conditions but it's difficult to get around
		// this elegantly. is we store strings for KeyboardDown events then,
		// because the keyboard is mostly numbers, converting them back from the
		// file will require a prefix of some sort to force it to look like a
		// string, rather than a float. that's probably a more ugly solution.
		//
		// any other solution requires altering the handcontroller
		// implementation which I don't want to do - the problem is caused here
		// and so should be mitigated here.
		//
		// likewise for KeyboardUp events. the handcontroller Handle() function
		// expects a nil argument for these events but we store the empty
		// string, instead of nil.
		if entry.event == ports.KeyboardDown {
			entry.value = rune(entry.value.(float32))
		} else if entry.event == ports.KeyboardUp {
			entry.value = nil
		}

		entry.frame, err = strconv.Atoi(toks[fieldFrame])
		if err != nil {
			return nil, curated.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldFrame+1], fieldSep)))
		}

		// assuming that frames are listed in order in the file. update
		// endFrame with the most recent frame every time
		plb.endFrame = entry.frame

		entry.scanline, err = strconv.Atoi(toks[fieldScanline])
		if err != nil {
			return nil, curated.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldScanline+1], fieldSep)))
		}

		entry.horizpos, err = strconv.Atoi(toks[fieldHorizPos])
		if err != nil {
			return nil, curated.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldHorizPos+1], fieldSep)))
		}

		entry.hash = toks[fieldHash]

		// add new entry to list of events in the correct playback sequence
		plb.sequence = append(plb.sequence, entry)
	}

	return plb, nil
}

// parse value entry as best we can. the theory here is that there is no
// intersection between the sets of allowed values. a bool doesn't look like a
// float which doesn't look like an int. if the value looks like none of those
// things then we can return the original string unchanged.
func parseEventData(value string) ports.EventData {
	var err error

	// the order of these conversions is important. ParseBool will interpret
	// "0" or "1" as false and true. we want to treat these value as ints or
	// floats (a float of 0.0 will be written as 0) so we MUST try converting
	// to those types first

	var f float64
	f, err = strconv.ParseFloat(value, 32)
	if err == nil {
		return float32(f)
	}

	var b bool
	b, err = strconv.ParseBool(value)
	if err == nil {
		return b
	}

	return value
}

// AttachToVCS attaches the playback instance (an implementation of the
// playback interface) to all the ports of the VCS, including the panel.
//
// Note that this will reset the VCS.
func (plb *Playback) AttachToVCS(vcs *hardware.VCS) error {
	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return curated.Errorf("playback: no playback hardware available")
	}
	plb.vcs = vcs

	var err error

	// we want the machine in a known state. the easiest way to do this is to
	// reset the hardware preferences
	err = vcs.Prefs.Reset()
	if err != nil {
		return curated.Errorf("playback: %v", err)
	}

	// validate header. keep it simple and disallow any difference in tv
	// specification. some combinations may work but there's no compelling
	// reason to figure that out just now.
	if plb.vcs.TV.GetReqSpecID() != plb.TVSpec {
		return curated.Errorf("playback: recording was made with the %s TV spec. trying to playback with a TV spec of %s.", plb.TVSpec, vcs.TV.GetReqSpecID())
	}

	plb.digest, err = digest.NewVideo(plb.vcs.TV)
	if err != nil {
		return curated.Errorf("playback: %v", err)
	}

	// attach playback to all vcs ports
	vcs.RIOT.Ports.AttachPlayback(plb)

	return nil
}

// Sentinal error returned by GetPlayback if a hash error is encountered.
const (
	PlaybackHashError = "playback: hash error [line %d]"
)

// GetPlayback returns an event and source portID for an event occurring at the
// current TV frame/scanline/horizpos.
func (plb *Playback) GetPlayback() (ports.PortID, ports.Event, ports.EventData, error) {
	// we've reached the end of the list of events for this id
	if plb.seqCt >= len(plb.sequence) {
		return ports.NoPortID, ports.NoEvent, nil, nil
	}

	// get current state of the television
	frame := plb.vcs.TV.GetState(signal.ReqFramenum)
	scanline := plb.vcs.TV.GetState(signal.ReqScanline)
	horizpos := plb.vcs.TV.GetState(signal.ReqHorizPos)

	// compare current state with the recording
	entry := plb.sequence[plb.seqCt]
	if frame == entry.frame && scanline == entry.scanline && horizpos == entry.horizpos {
		plb.seqCt++
		if entry.hash != plb.digest.Hash() {
			return ports.NoPortID, ports.NoEvent, nil, curated.Errorf(PlaybackHashError, entry.line)
		}
		return entry.portID, entry.event, entry.value, nil
	}

	// next event does not match
	return ports.NoPortID, ports.NoEvent, nil, nil
}
