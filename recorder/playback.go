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
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/digest"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

type playbackEntry struct {
	portID plugging.PortID
	event  ports.Event
	data   ports.EventData
	coords coords.TelevisionCoords
	hash   string

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
	currFrame := plb.digest.GetCoords().Frame
	return fmt.Sprintf("%d/%d (%.1f%%)", currFrame, plb.endFrame, 100*(float64(currFrame)/float64(plb.endFrame)))
}

// EndFrame returns true if emulation has gone past the last frame of the
// playback.
func (plb Playback) EndFrame() (bool, error) {
	currFrame := plb.digest.GetCoords().Frame
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
	buffer, err := io.ReadAll(tf)
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

		// create a new playbackEntry and convert tokens accordingly any errors in the transcript causes failure
		entry := playbackEntry{line: i + 1}

		// get PortID
		n, err := strconv.Atoi(toks[fieldPortID])
		if err != nil {
			entry.portID = plugging.PortID(toks[fieldPortID])
		} else {
			// support for playback file versions before v1.2
			switch n {
			case -1:
				entry.portID = plugging.PortUnplugged
			case 1:
				entry.portID = plugging.PortLeftPlayer
			case 2:
				entry.portID = plugging.PortRightPlayer
			case 3:
				entry.portID = plugging.PortPanel
			default:
				return nil, curated.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldPortID+1], fieldSep)))
			}
		}

		// no need to convert event field
		entry.event = ports.Event(toks[fieldEvent])

		// entry data is of ports.EventDataPlayback type. The ports
		// implementation will handle parsing of this type.
		entry.data = ports.EventDataPlayback(toks[fieldEventData])

		entry.coords.Frame, err = strconv.Atoi(toks[fieldFrame])
		if err != nil {
			return nil, curated.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldFrame+1], fieldSep)))
		}

		// assuming that frames are listed in order in the file. update
		// endFrame with the most recent frame every time
		plb.endFrame = entry.coords.Frame

		entry.coords.Scanline, err = strconv.Atoi(toks[fieldScanline])
		if err != nil {
			return nil, curated.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldScanline+1], fieldSep)))
		}

		entry.coords.Clock, err = strconv.Atoi(toks[fieldClock])
		if err != nil {
			return nil, curated.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldClock+1], fieldSep)))
		}

		entry.hash = toks[fieldHash]

		// add new entry to list of events in the correct playback sequence
		plb.sequence = append(plb.sequence, entry)
	}

	return plb, nil
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
	err = vcs.RIOT.Ports.AttachPlayback(plb)
	if err != nil {
		return curated.Errorf("playback: %v", err)
	}

	return nil
}

// Sentinal error returned by GetPlayback if a hash error is encountered.
const (
	PlaybackHashError = "playback: unexpected input at line %d (frame %d)"
)

// GetPlayback returns an event and source PortID for an event occurring at the
// current TV frame/scanline/clock.
func (plb *Playback) GetPlayback() (plugging.PortID, ports.Event, ports.EventData, error) {
	// we've reached the end of the list of events for this id
	if plb.seqCt >= len(plb.sequence) {
		return plugging.PortUnplugged, ports.NoEvent, nil, nil
	}

	// get current state of the television
	curr := plb.vcs.TV.GetCoords()

	// compare current state with the recording
	entry := plb.sequence[plb.seqCt]
	if coords.Equal(entry.coords, curr) {
		plb.seqCt++
		if entry.hash != plb.digest.Hash() {
			return plugging.PortUnplugged, ports.NoEvent, nil, curated.Errorf(PlaybackHashError, entry.line, curr.Frame)
		}
		return entry.portID, entry.event, entry.data, nil
	}

	// next event does not match
	return plugging.PortUnplugged, ports.NoEvent, nil, nil
}
