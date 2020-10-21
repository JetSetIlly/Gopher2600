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
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/digest"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/setup"
)

const videoEntryID = "video"

const (
	videoFieldCartName int = iota
	videoFieldCartMapping
	videoFieldTVtype
	videoFieldNumFrames
	videoFieldState
	videoFieldStateOptions
	videoFieldStateFile
	videoFieldDigest
	videoFieldNotes
	numVideoFields
)

// VideoRegression is the simplest regression type. it works by running the
// emulation for N frames and the video recorded at that point. Regression
// passes if subsequenct runs produce the same video value.
type VideoRegression struct {
	CartLoad     cartridgeloader.Loader
	TVtype       string
	NumFrames    int
	State        StateType
	stateOptions string
	stateFile    string
	Notes        string
	digest       string
}

func deserialiseVideoEntry(fields database.SerialisedEntry) (database.Entry, error) {
	reg := &VideoRegression{}

	// basic sanity check
	if len(fields) > numVideoFields {
		return nil, curated.Errorf("video: too many fields")
	}
	if len(fields) < numVideoFields {
		return nil, curated.Errorf("video: too few fields")
	}

	// string fields need no conversion
	reg.CartLoad.Filename = fields[videoFieldCartName]
	reg.CartLoad.Mapping = fields[videoFieldCartMapping]
	reg.TVtype = fields[videoFieldTVtype]
	reg.digest = fields[videoFieldDigest]
	reg.Notes = fields[videoFieldNotes]

	var err error

	// convert number of frames field
	reg.NumFrames, err = strconv.Atoi(fields[videoFieldNumFrames])
	if err != nil {
		return nil, curated.Errorf("video: invalid numFrames field [%s]", fields[videoFieldNumFrames])
	}

	// handle state field
	switch fields[videoFieldState] {
	case "":
		reg.State = StateNone
	case "TV":
		reg.State = StateTV
	case "PORTS":
		reg.State = StatePorts
	case "TIMER":
		reg.State = StateTimer
	case "CPU":
		reg.State = StateCPU
	default:
		return nil, curated.Errorf("video: invalid state field [%s]", fields[videoFieldState])
	}

	// state options
	reg.stateOptions = fields[videoFieldStateOptions]

	// and state file field
	if fields[videoFieldStateFile] != "" {
		if reg.State == StateNone {
			return nil, curated.Errorf("video: invalid state file field: no state type specifier")
		}
		reg.stateFile = fields[videoFieldStateFile]
	}

	return reg, nil
}

// ID implements the database.Entry interface.
func (reg VideoRegression) ID() string {
	return videoEntryID
}

// String implements the database.Entry interface.
func (reg VideoRegression) String() string {
	s := strings.Builder{}

	state := ""
	switch reg.State {
	case StateNone:
		state = ""
	case StateTV:
		state = " [TV state]"
	case StatePorts:
		state = " [ports state]"
	case StateTimer:
		state = " [timer state]"
	case StateCPU:
		state = " [cpu state]"
	default:
		state = " [with state]"
	}

	s.WriteString(fmt.Sprintf("[%s] %s [%s] frames=%d%s", reg.ID(), reg.CartLoad.ShortName(), reg.TVtype, reg.NumFrames, state))
	if reg.Notes != "" {
		s.WriteString(fmt.Sprintf(" [%s]", reg.Notes))
	}
	return s.String()
}

// Serialise implements the database.Entry interface.
func (reg *VideoRegression) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			reg.CartLoad.Filename,
			reg.CartLoad.Mapping,
			reg.TVtype,
			strconv.Itoa(reg.NumFrames),
			reg.State.String(),
			reg.stateOptions,
			reg.stateFile,
			reg.digest,
			reg.Notes,
		},
		nil
}

// CleanUp implements the database.Entry interface.
func (reg VideoRegression) CleanUp() error {
	err := os.Remove(reg.stateFile)
	if _, ok := err.(*os.PathError); ok {
		return nil
	}
	return err
}

// regress implements the regression.Regressor interface.
func (reg *VideoRegression) regress(newRegression bool, output io.Writer, msg string, skipCheck func() bool) (_ bool, _ string, rerr error) {
	output.Write([]byte(msg))

	// create headless television. we'll use this to initialise the digester
	tv, err := television.NewTelevision(reg.TVtype)
	if err != nil {
		return false, "", curated.Errorf("video: %v", err)
	}
	defer tv.End()

	dig, err := digest.NewVideo(tv)
	if err != nil {
		return false, "", curated.Errorf("video: %v", err)
	}

	// create VCS and attach cartridge
	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return false, "", curated.Errorf("video: %v", err)
	}

	// we want the machine in a known state. the easiest way to do this is to
	// reset the hardware preferences
	err = vcs.Prefs.Reset()
	if err != nil {
		return false, "", curated.Errorf("video: %v", err)
	}

	err = setup.AttachCartridge(vcs, reg.CartLoad)
	if err != nil {
		return false, "", curated.Errorf("video: %v", err)
	}

	// list of state information. we'll either save this in the event of
	// newRegression being true; or we'll use it to compare to the entries in
	// the specified state file
	state := make([]string, 0, 1024)

	// add the starting state of the tv
	switch reg.State {
	case StateTV:
		state = append(state, tv.String())
	case StatePorts:
		state = append(state, vcs.RIOT.Ports.String())
	case StateTimer:
		state = append(state, vcs.RIOT.Timer.String())
	case StateCPU:
		state = append(state, vcs.CPU.String())
	}

	// display ticker for progress meter
	dur, _ := time.ParseDuration("1s")
	tck := time.NewTicker(dur)

	// run emulation
	err = vcs.RunForFrameCount(reg.NumFrames, func(frame int) (bool, error) {
		if skipCheck() {
			return false, curated.Errorf(regressionSkipped)
		}

		// display progress meter every 1 second
		select {
		case <-tck.C:
			output.Write([]byte(fmt.Sprintf("\r%s [%d/%d (%.1f%%)]", msg, frame, reg.NumFrames, 100*(float64(frame)/float64(reg.NumFrames)))))
		default:
		}

		// store state. StateTV stores every video cycle. other State types
		// can (should?) choose to only store state if it is different to the
		// previous entry
		switch reg.State {
		case StateTV:
			state = append(state, tv.String())
		case StatePorts:
			state = append(state, vcs.RIOT.Ports.String())
		case StateTimer:
			state = append(state, vcs.RIOT.Timer.String())
		case StateCPU:
			state = append(state, vcs.CPU.String())
		}

		return true, nil
	})

	if err != nil {
		return false, "", curated.Errorf("video: %v", err)
	}

	if newRegression {
		reg.digest = dig.Hash()

		if reg.State != StateNone {
			// create a unique filename
			reg.stateFile, err = uniqueFilename("state", reg.CartLoad)
			if err != nil {
				return false, "", curated.Errorf("video: %v", err)
			}

			// check that the filename is unique
			nf, _ := os.Open(reg.stateFile)

			// no need to bother with returned error. nf tells us everything we
			// need
			if nf != nil {
				return false, "", curated.Errorf("video: state recording file already exists (%s)", reg.stateFile)
			}
			nf.Close()

			// create new file
			nf, err = os.Create(reg.stateFile)
			if err != nil {
				return false, "", curated.Errorf("video: error creating state recording file: %v", err)
			}
			defer func() {
				err := nf.Close()
				if err != nil {
					rerr = curated.Errorf("video: error creating state recording file: %v", err)
				}
			}()

			for i := range state {
				s := fmt.Sprintf("%s\n", state[i])
				if n, err := nf.WriteString(s); err != nil || len(s) != n {
					return false, "", curated.Errorf("video: error writing state recording file: %v", err)
				}
			}
		}

		// this is a new regression entry so we don't need to do the comparison
		// stage so we return early
		return true, "", nil
	}

	// only for replay of existing regression entries. compare new state
	// tracking with recorded state tracking
	if reg.State != StateNone {
		nf, err := os.Open(reg.stateFile)
		if err != nil {
			return false, "", curated.Errorf("video: old state recording file not present (%s)", reg.stateFile)
		}
		defer nf.Close()

		reader := bufio.NewReader(nf)

		for i := range state {
			s, _ := reader.ReadString('\n')
			s = strings.TrimRight(s, "\n")

			// ignore blank lines
			if s == "" {
				continue
			}

			if s != state[i] {
				failm := fmt.Sprintf("state mismatch line %d: expected %s (%s)", i, s, state[i])
				return false, failm, nil
			}
		}

		// check that we've consumed all the lines in the recorded state file
		_, err = reader.ReadString('\n')
		if err == nil || err != io.EOF {
			failm := "unexpected end of state. entries remaining in recorded state file"
			return false, failm, nil
		}
	}

	if dig.Hash() != reg.digest {
		return false, "digest mismatch", nil
	}

	return true, "", nil
}
