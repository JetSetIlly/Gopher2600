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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/digest"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/recorder"
)

const playbackEntryType = "playback"

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

	// do not check screen digests during playback
	ignoreDigest bool
}

func deserialisePlaybackEntry(fields database.SerialisedEntry) (database.Entry, error) {
	reg := &PlaybackRegression{}

	// basic sanity check
	if len(fields) > numPlaybackFields {
		return nil, fmt.Errorf("playback: too many fields")
	}
	if len(fields) < numPlaybackFields {
		return nil, fmt.Errorf("playback: too few fields")
	}

	// string fields need no conversion
	reg.Script = fields[playbackFieldScript]
	reg.Notes = fields[playbackFieldNotes]

	return reg, nil
}

// EntryType implements the database.Entry interface.
func (reg PlaybackRegression) EntryType() string {
	return playbackEntryType
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
	if err != nil {
		var pathError *os.PathError
		if errors.As(err, &pathError) {
			return nil
		}
	}
	return err
}

// String implements the regression.Regressor interface.
func (reg PlaybackRegression) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("[%s] %s", reg.EntryType(), filepath.Base(reg.Script)))
	if reg.Notes != "" {
		s.WriteString(fmt.Sprintf(" [%s]", reg.Notes))
	}
	return s.String()
}

// concurrentSafe implements the regression.Regressor interface.
func (reg *PlaybackRegression) concurrentSafe() bool {
	return true
}

// redux implements the regression.Regressor interface.
func (reg *PlaybackRegression) redux(messages io.Writer, tag string) (Regressor, error) {
	old := *reg
	reg.ignoreDigest = true
	defer func() {
		reg.ignoreDigest = false
	}()
	return &old, reg.regress(true, messages, tag)
}

// regress implements the regression.Regressor interface.
func (reg *PlaybackRegression) regress(newRegression bool, messages io.Writer, tag string) (rerr error) {
	messages.Write([]byte(tag))

	plb, err := recorder.NewPlayback(reg.Script, reg.ignoreDigest)
	if err != nil {
		return fmt.Errorf("playback: %w", err)
	}

	tv, err := television.NewTelevision(plb.TVSpec)
	if err != nil {
		return fmt.Errorf("playback: %w", err)
	}
	defer tv.End()
	tv.SetFPSCap(false)

	_, err = digest.NewVideo(tv)
	if err != nil {
		return fmt.Errorf("playback: %w", err)
	}

	vcs, err := hardware.NewVCS(environment.MainEmulation, tv, nil, nil)
	if err != nil {
		return fmt.Errorf("playback: %w", err)
	}

	// for playback regression to work correctly we want the VCS to be a known
	// starting state. this will be handled in the playback.AttachToVCS
	// function according to the current features of the recorder package and
	// the saved script

	err = plb.AttachToVCSInput(vcs)
	if err != nil {
		return fmt.Errorf("playback: %w", err)
	}

	// new cartridge loader using the information found in the playback file
	cartload, err := cartridgeloader.NewLoaderFromFilename(plb.Cartridge, "AUTO", "AUTO", nil)
	if err != nil {
		return fmt.Errorf("playback: %w", err)
	}
	defer cartload.Close()

	// check hash of cartridge before continuing
	if cartload.HashSHA1 != plb.Hash {
		return fmt.Errorf("playback: unexpected hash")
	}

	if newRegression {
		var err error
		reg.Script, err = uniqueFilename("playback", cartload.Name)
		if err != nil {
			return fmt.Errorf("playback: %w", err)
		}
		rec, err := recorder.NewRecorder(reg.Script, vcs)
		if err != nil {
			return fmt.Errorf("playback: %w", err)
		}
		defer func() {
			err := rec.End()
			if err != nil {
				rerr = err
			}
		}()
	}

	// not using setup.AttachCartridge. if the playback was recorded with setup
	// changes the events will have been copied into the playback script and
	// will be applied that way
	err = vcs.AttachCartridge(cartload)
	if err != nil {
		return fmt.Errorf("playback: %w", err)
	}

	// prepare ticker for progress meter
	dur, _ := time.ParseDuration("1s")
	tck := time.NewTicker(dur)

	// run emulation
	err = vcs.Run(func() (govern.State, error) {
		// if the CPU is in the KIL state then the test will never end normally
		if vcs.CPU.Killed {
			return govern.Ending, fmt.Errorf("CPU in KIL state")
		}

		hasEnded, err := plb.EndFrame()
		if err != nil {
			return govern.Ending, fmt.Errorf("playback: %w", err)
		}
		if hasEnded {
			return govern.Ending, fmt.Errorf("playback: ended unexpectedly")
		}

		// display progress meter every 1 second
		select {
		case <-tck.C:
			messages.Write([]byte(fmt.Sprintf("\r%s [%s]", tag, plb)))
		default:
		}
		return govern.Running, nil
	})

	if err != nil {
		if errors.Is(err, ports.PowerOff) {
			// PowerOff is okay and is to be expected
		} else if errors.Is(err, recorder.PlaybackHashError) {
			// PlaybackHashError means that a screen digest somewhere in the
			// playback script did not work. filter error and return false to
			// indicate failure
			coords := tv.GetCoords()
			return fmt.Errorf("%w: at fr=%d, sl=%d, cl=%d", err, coords.Frame, coords.Scanline, coords.Clock)
		} else {
			return fmt.Errorf("playback: %w", err)
		}
	}

	return nil
}
