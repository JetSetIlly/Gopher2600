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

package regression

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/digest"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/television"
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
		return nil, errors.New(errors.RegressionPlaybackError, "too many fields")
	}
	if len(fields) < numPlaybackFields {
		return nil, errors.New(errors.RegressionPlaybackError, "too few fields")
	}

	// string fields need no conversion
	reg.Script = fields[playbackFieldScript]
	reg.Notes = fields[playbackFieldNotes]

	return reg, nil
}

// ID implements the database.Entry interface
func (reg PlaybackRegression) ID() string {
	return playbackEntryID
}

// String implements the database.Entry interface
func (reg PlaybackRegression) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("[%s] %s", reg.ID(), path.Base(reg.Script)))
	if reg.Notes != "" {
		s.WriteString(fmt.Sprintf(" [%s]", reg.Notes))
	}
	return s.String()
}

// Serialise implements the database.Entry interface
func (reg *PlaybackRegression) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			reg.Script,
			reg.Notes,
		},
		nil
}

// CleanUp implements the database.Entry interface
func (reg PlaybackRegression) CleanUp() error {
	err := os.Remove(reg.Script)
	if _, ok := err.(*os.PathError); ok {
		return nil
	}
	return err
}

// regress implements the regression.Regressor interface
func (reg *PlaybackRegression) regress(newRegression bool, output io.Writer, msg string) (bool, string, error) {
	output.Write([]byte(msg))

	plb, err := recorder.NewPlayback(reg.Script)
	if err != nil {
		return false, "", errors.New(errors.RegressionPlaybackError, err)
	}

	tv, err := television.NewTelevision(plb.TVSpec)
	if err != nil {
		return false, "", errors.New(errors.RegressionPlaybackError, err)
	}
	defer tv.End()

	_, err = digest.NewVideo(tv)
	if err != nil {
		return false, "", errors.New(errors.RegressionPlaybackError, err)
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return false, "", errors.New(errors.RegressionPlaybackError, err)
	}

	err = plb.AttachToVCS(vcs)
	if err != nil {
		return false, "", errors.New(errors.RegressionPlaybackError, err)
	}

	// not using setup.AttachCartridge. if the playback was recorded with setup
	// changes the events will have been copied into the playback script and
	// will be applied that way
	err = vcs.AttachCartridge(plb.CartLoad)
	if err != nil {
		return false, "", errors.New(errors.RegressionPlaybackError, err)
	}

	// prepare ticker for progress meter
	dur, _ := time.ParseDuration("1s")
	tck := time.NewTicker(dur)

	// run emulation
	err = vcs.Run(func() (bool, error) {
		hasEnded, err := plb.EndFrame()
		if err != nil {
			return false, errors.New(errors.RegressionPlaybackError, err)
		}
		if hasEnded {
			return false, errors.New(errors.RegressionPlaybackError, "playback has not ended as expected")
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
		if !errors.IsAny(err) {
			return false, "", errors.New(errors.RegressionPlaybackError, err)
		}

		switch err.(errors.AtariError).Message {
		// the PowerOff error is expected. if we receive it then that means
		// the regression test has succeeded
		case errors.PowerOff:
			break

		// PlaybackHashError means that a screen digest somewhere in the
		// playback script did not work. filter error and return false to
		// indicate failure
		case errors.PlaybackHashError:
			fr, _ := tv.GetState(television.ReqFramenum)
			sl, _ := tv.GetState(television.ReqScanline)
			hp, _ := tv.GetState(television.ReqHorizPos)
			failm := fmt.Sprintf("%v: at fr=%d, sl=%d, hp=%d", err, fr, sl, hp)
			return false, failm, nil

		default:
			return false, "", errors.New(errors.RegressionPlaybackError, err)
		}

	}

	// if this is a new regression we want to store the script in the
	// regressionScripts directory
	if newRegression {
		// create a unique filename
		newScript, err := uniqueFilename("playback", plb.CartLoad)
		if err != nil {
			return false, "", errors.New(errors.RegressionPlaybackError, err)
		}

		// check that the filename is unique
		nf, _ := os.Open(newScript)
		// no need to bother with returned error. nf tells us everything we
		// need
		if nf != nil {
			msg := fmt.Sprintf("script already exists (%s)", newScript)
			return false, "", errors.New(errors.RegressionPlaybackError, msg)
		}
		nf.Close()

		// create new file
		nf, err = os.Create(newScript)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, "", errors.New(errors.RegressionPlaybackError, msg)
		}
		defer nf.Close()

		// open old file
		of, err := os.Open(reg.Script)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, "", errors.New(errors.RegressionPlaybackError, msg)
		}
		defer of.Close()

		// copy old file to new file
		_, err = io.Copy(nf, of)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, "", errors.New(errors.RegressionPlaybackError, msg)
		}

		// update script name in regression type
		reg.Script = newScript
	}

	return true, "", nil
}
