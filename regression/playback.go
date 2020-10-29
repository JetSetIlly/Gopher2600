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

package regression

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/digest"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/recorder"
)

const playbackEntryID = "playback"

const (
	playbackFieldScript int = iota
	playbackFieldNotes
	numPlaybackFields
)

// PlaybackRegression represents a regression type that processes a VCS
// recording. playback regressions can take a while to run because by their
// nature they extend over many frames - many more than is typical with the
// FrameRegression type.
type PlaybackRegression struct {
	Script string
	Notes  string
}

func deserialisePlaybackEntry(fields database.SerialisedEntry) (database.Entry, error) {
	reg := &PlaybackRegression{}

	// basic sanity check
	if len(fields) > numPlaybackFields {
		return nil, curated.Errorf("playback: too many fields")
	}
	if len(fields) < numPlaybackFields {
		return nil, curated.Errorf("playback: too few fields")
	}

	// string fields need no conversion
	reg.Script = fields[playbackFieldScript]
	reg.Notes = fields[playbackFieldNotes]

	return reg, nil
}

// ID implements the database.Entry interface.
func (reg PlaybackRegression) ID() string {
	return playbackEntryID
}

// String implements the database.Entry interface.
func (reg PlaybackRegression) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("[%s] %s", reg.ID(), path.Base(reg.Script)))
	if reg.Notes != "" {
		s.WriteString(fmt.Sprintf(" [%s]", reg.Notes))
	}
	return s.String()
}

// Serialise implements the database.Entry interface.
func (reg *PlaybackRegression) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			reg.Script,
			reg.Notes,
		},
		nil
}

// CleanUp implements the database.Entry interface.
func (reg PlaybackRegression) CleanUp() error {
	err := os.Remove(reg.Script)
	if _, ok := err.(*os.PathError); ok {
		return nil
	}
	return err
}

// regress implements the regression.Regressor interface.
func (reg *PlaybackRegression) regress(newRegression bool, output io.Writer, msg string, skipCheck func() bool) (_ bool, _ string, rerr error) {
	output.Write([]byte(msg))

	plb, err := recorder.NewPlayback(reg.Script)
	if err != nil {
		return false, "", curated.Errorf("playback: %v", err)
	}

	tv, err := television.NewTelevision(plb.TVSpec)
	if err != nil {
		return false, "", curated.Errorf("playback: %v", err)
	}
	defer tv.End()

	_, err = digest.NewVideo(tv)
	if err != nil {
		return false, "", curated.Errorf("playback: %v", err)
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return false, "", curated.Errorf("playback: %v", err)
	}

	// for playback regression to work correctly we want the VCS to be a known
	// starting state. this will be handled in the playback.AttachToVCS
	// function according to the current features of the recorder package and
	// the saved script

	err = plb.AttachToVCS(vcs)
	if err != nil {
		return false, "", curated.Errorf("playback: %v", err)
	}

	// not using setup.AttachCartridge. if the playback was recorded with setup
	// changes the events will have been copied into the playback script and
	// will be applied that way
	err = vcs.AttachCartridge(plb.CartLoad)
	if err != nil {
		return false, "", curated.Errorf("playback: %v", err)
	}

	// prepare ticker for progress meter
	dur, _ := time.ParseDuration("1s")
	tck := time.NewTicker(dur)

	// run emulation
	err = vcs.Run(func() (bool, error) {
		if skipCheck() {
			return false, curated.Errorf(regressionSkipped)
		}

		hasEnded, err := plb.EndFrame()
		if err != nil {
			return false, curated.Errorf("playback: %v", err)
		}
		if hasEnded {
			return false, curated.Errorf("playback: ended unexpectedly")
		}

		// display progress meter every 1 second
		select {
		case <-tck.C:
			output.Write([]byte(fmt.Sprintf("\r%s [%s]", msg, plb)))
		default:
		}
		return true, nil
	})

	if err != nil {
		if curated.Has(err, ports.PowerOff) {
			// PowerOff is okay and is to be expected
		} else if curated.Has(err, recorder.PlaybackHashError) {
			// PlaybackHashError means that a screen digest somewhere in the
			// playback script did not work. filter error and return false to
			// indicate failure
			fr := tv.GetState(signal.ReqFramenum)
			sl := tv.GetState(signal.ReqScanline)
			hp := tv.GetState(signal.ReqHorizPos)
			failm := fmt.Sprintf("%v: at fr=%d, sl=%d, hp=%d", err, fr, sl, hp)
			return false, failm, nil
		} else {
			return false, "", curated.Errorf("playback: %v", err)
		}
	}

	// if this is a new regression we want to store the script in the
	// regressionScripts directory
	if newRegression {
		// create a unique filename
		newScript, err := uniqueFilename("playback", plb.CartLoad)
		if err != nil {
			return false, "", curated.Errorf("playback: %v", err)
		}

		// check that the filename is unique
		nf, _ := os.Open(newScript)
		// no need to bother with returned error. nf tells us everything we
		// need
		if nf != nil {
			return false, "", curated.Errorf("playback: script already exists (%s)", newScript)
		}
		nf.Close()

		// create new file
		nf, err = os.Create(newScript)
		if err != nil {
			return false, "", curated.Errorf("playback: while copying playback script: %v", err)
		}
		defer func() {
			err := nf.Close()
			if err != nil {
				rerr = curated.Errorf("playback: while copying playback script: %v", err)
			}
		}()

		// open old file
		of, err := os.Open(reg.Script)
		if err != nil {
			return false, "", curated.Errorf("playback: while copying playback script: %v", err)
		}
		defer of.Close()

		// copy old file to new file
		_, err = io.Copy(nf, of)
		if err != nil {
			return false, "", curated.Errorf("playback: while copying playback script: %v", err)
		}

		// update script name in regression type
		reg.Script = newScript
	}

	return true, "", nil
}
