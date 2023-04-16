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
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/digest"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

type playbackEntry struct {
	event ports.TimedInputEvent
	hash  string

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
//
// The returned playback must be attached to the VCS input system (with
// AttachToVCSInput() function) for it it to be useful.
//
// The integrityCheck flag should be true in most instances.
func NewPlayback(transcript string, checkROM bool) (*Playback, error) {
	var err error

	plb := &Playback{
		transcript: transcript,
		sequence:   make([]playbackEntry, 0),
	}

	tf, err := os.Open(transcript)
	if err != nil {
		return nil, fmt.Errorf("playback: %w", err)
	}
	buffer, err := io.ReadAll(tf)
	if err != nil {
		return nil, fmt.Errorf("playback: %w", err)
	}
	err = tf.Close()
	if err != nil {
		return nil, fmt.Errorf("playback: %w", err)
	}

	// convert file contents to an array of lines
	lines := strings.Split(string(buffer), "\n")

	// read header and perform validation checks
	err = plb.readHeader(lines, checkROM)
	if err != nil {
		return nil, err
	}

	// loop through transcript and divide events according to the first field
	// (the peripheral ID)
	for i := numHeaderLines; i < len(lines)-1; i++ {
		toks := strings.Split(lines[i], fieldSep)

		// ignore lines that don't have enough fields
		if len(toks) != numFields {
			return nil, fmt.Errorf("playback: expected %d fields at line %d", numFields, i+1)
		}

		// create a new playbackEntry and convert tokens accordingly any errors in the transcript causes failure
		entry := playbackEntry{line: i + 1}

		// get PortID
		n, err := strconv.Atoi(toks[fieldPortID])
		if err != nil {
			entry.event.Port = plugging.PortID(toks[fieldPortID])
		} else {
			// support for playback file versions before v1.2
			switch n {
			case -1:
				entry.event.Port = plugging.PortUnplugged
			case 1:
				entry.event.Port = plugging.PortLeft
			case 2:
				entry.event.Port = plugging.PortRight
			case 3:
				entry.event.Port = plugging.PortPanel
			default:
				return nil, fmt.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldPortID+1], fieldSep)))
			}
		}

		// no need to convert event field
		entry.event.Ev = ports.Event(toks[fieldEvent])

		// entry data is of ports.EventDataPlayback type. The ports
		// implementation will handle parsing of this type.
		entry.event.D = ports.EventDataPlayback(toks[fieldEventData])

		entry.event.Time.Frame, err = strconv.Atoi(toks[fieldFrame])
		if err != nil {
			return nil, fmt.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldFrame+1], fieldSep)))
		}

		// assuming that frames are listed in order in the file. update
		// endFrame with the most recent frame every time
		plb.endFrame = entry.event.Time.Frame

		entry.event.Time.Scanline, err = strconv.Atoi(toks[fieldScanline])
		if err != nil {
			return nil, fmt.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldScanline+1], fieldSep)))
		}

		entry.event.Time.Clock, err = strconv.Atoi(toks[fieldClock])
		if err != nil {
			return nil, fmt.Errorf("playback: %s line %d, col %d", err, i+1, len(strings.Join(toks[:fieldClock+1], fieldSep)))
		}

		entry.hash = toks[fieldHash]

		// add new entry to list of events in the correct playback sequence
		plb.sequence = append(plb.sequence, entry)
	}

	return plb, nil
}

// AttachToVCSInput attaches the playback instance (an implementation of the
// playback interface) to all the input system of the VCS.
//
// Note that the VCS instance will be normalised as a result of this call.
func (plb *Playback) AttachToVCSInput(vcs *hardware.VCS) error {
	// check we're working with correct information
	if vcs == nil || vcs.TV == nil {
		return fmt.Errorf("playback: no playback hardware available")
	}
	plb.vcs = vcs

	var err error

	// we want the machine in a known state. the easiest way to do this is to
	// default the hardware preferences
	vcs.Env.Normalise()

	// validate header. keep it simple and disallow any difference in tv
	// specification. some combinations may work but there's no compelling
	// reason to figure that out just now.
	if plb.vcs.TV.GetReqSpecID() != plb.TVSpec {
		return fmt.Errorf("playback: recording was made with the %s TV spec. trying to playback with a TV spec of %s.", plb.TVSpec, vcs.TV.GetReqSpecID())
	}

	plb.digest, err = digest.NewVideo(plb.vcs.TV)
	if err != nil {
		return fmt.Errorf("playback: %w", err)
	}

	// attach playback to all VCS Input system
	vcs.Input.AttachPlayback(plb)

	return nil
}

// sentinal error returned by GetPlayback() if a hash error is encountered.
var PlaybackHashError = errors.New("unexpected input")

// GetPlayback returns an event and source PortID for an event occurring at the
// current TV frame/scanline/clock.
func (plb *Playback) GetPlayback() (ports.TimedInputEvent, error) {
	// get current state of the television
	c := plb.vcs.TV.GetCoords()

	// we've reached the end of the list of events for this id
	if plb.seqCt >= len(plb.sequence) {
		return ports.TimedInputEvent{
			Time: c,
			InputEvent: ports.InputEvent{
				Port: plugging.PortUnplugged,
				Ev:   ports.NoEvent,
			},
		}, nil
	}

	// compare current state with the recording
	entry := plb.sequence[plb.seqCt]
	if coords.Equal(entry.event.Time, c) {
		plb.seqCt++
		if entry.hash != plb.digest.Hash() {
			return ports.TimedInputEvent{
				Time: c,
				InputEvent: ports.InputEvent{
					Port: plugging.PortUnplugged,
					Ev:   ports.NoEvent,
				},
			}, fmt.Errorf("playback: %w: line %d (frame %d)", PlaybackHashError, entry.line, c.Frame)
		}
		return entry.event, nil
	}

	// next event does not match
	return ports.TimedInputEvent{
		Time: c,
		InputEvent: ports.InputEvent{
			Port: plugging.PortUnplugged,
			Ev:   ports.NoEvent,
		},
	}, nil
}
