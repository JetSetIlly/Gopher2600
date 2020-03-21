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
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/digest"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/setup"
	"github.com/jetsetilly/gopher2600/television"
)

const digestEntryID = "digest"

const (
	digestFieldMode int = iota
	digestFieldCartName
	digestFieldCartFormat
	digestFieldTVtype
	digestFieldNumFrames
	digestFieldState
	digestFieldDigest
	digestFieldNotes
	numDigestFields
)

// DigestRegression is the simplest regression type. it works by running the
// emulation for N frames and the digest recorded at that point. Regression
// passes if subsequenct runs produce the same digest value
type DigestRegression struct {
	Mode      DigestMode
	CartLoad  cartridgeloader.Loader
	TVtype    string
	NumFrames int
	State     bool
	stateFile string
	Notes     string
	digest    string
}

func deserialiseDigestEntry(fields database.SerialisedEntry) (database.Entry, error) {
	reg := &DigestRegression{}

	// basic sanity check
	if len(fields) > numDigestFields {
		return nil, errors.New(errors.RegressionDigestError, "too many fields")
	}
	if len(fields) < numDigestFields {
		return nil, errors.New(errors.RegressionDigestError, "too few fields")
	}

	// string fields need no conversion
	reg.CartLoad.Filename = fields[digestFieldCartName]
	reg.CartLoad.Format = fields[digestFieldCartFormat]
	reg.TVtype = fields[digestFieldTVtype]
	reg.digest = fields[digestFieldDigest]
	reg.Notes = fields[digestFieldNotes]

	var err error

	// parse mode field
	reg.Mode, err = ParseDigestMode(fields[digestFieldMode])
	if err != nil {
		return nil, errors.New(errors.RegressionDigestError, err)
	}

	// convert number of frames field
	reg.NumFrames, err = strconv.Atoi(fields[digestFieldNumFrames])
	if err != nil {
		msg := fmt.Sprintf("invalid numFrames field [%s]", fields[digestFieldNumFrames])
		return nil, errors.New(errors.RegressionDigestError, msg)
	}

	// handle state field
	if fields[digestFieldState] != "" {
		reg.State = true
		reg.stateFile = fields[digestFieldState]
	}

	return reg, nil
}

// ID implements the database.Entry interface
func (reg DigestRegression) ID() string {
	return digestEntryID
}

// String implements the database.Entry interface
func (reg DigestRegression) String() string {
	s := strings.Builder{}
	stateFile := ""
	if reg.State {
		stateFile = "[with state]"
	}

	s.WriteString(fmt.Sprintf("[%s/%s] %s [%s] frames=%d %s", reg.ID(), reg.Mode, reg.CartLoad.ShortName(), reg.TVtype, reg.NumFrames, stateFile))
	if reg.Notes != "" {
		s.WriteString(fmt.Sprintf(" [%s]", reg.Notes))
	}
	return s.String()
}

// Serialise implements the database.Entry interface
func (reg *DigestRegression) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			reg.Mode.String(),
			reg.CartLoad.Filename,
			reg.CartLoad.Format,
			reg.TVtype,
			strconv.Itoa(reg.NumFrames),
			reg.stateFile,
			reg.digest,
			reg.Notes,
		},
		nil
}

// CleanUp implements the database.Entry interface
func (reg DigestRegression) CleanUp() error {
	err := os.Remove(reg.stateFile)
	if _, ok := err.(*os.PathError); ok {
		return nil
	}
	return err
}

// regress implements the regression.Regressor interface
func (reg *DigestRegression) regress(newRegression bool, output io.Writer, msg string) (bool, string, error) {
	output.Write([]byte(msg))

	// create headless television. we'll use this to initialise the digester
	tv, err := television.NewTelevision(reg.TVtype)
	if err != nil {
		return false, "", errors.New(errors.RegressionDigestError, err)
	}
	defer tv.End()

	// decide on digest mode and create appropriate digester
	var dig digest.Digest

	switch reg.Mode {
	case DigestVideoOnly:
		dig, err = digest.NewVideo(tv)
		if err != nil {
			return false, "", errors.New(errors.RegressionDigestError, err)
		}

	case DigestAudioOnly:
		dig, err = digest.NewAudio(tv)
		if err != nil {
			return false, "", errors.New(errors.RegressionDigestError, err)
		}

	case DigestBoth:
		return false, "", errors.New(errors.RegressionDigestError, "video/audio digest not yet implemented")

	case DigestUndefined:
		return false, "", errors.New(errors.RegressionDigestError, fmt.Sprintf("undefined digest mode"))
	}

	// create VCS and attach cartridge
	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return false, "", errors.New(errors.RegressionDigestError, err)
	}

	err = setup.AttachCartridge(vcs, reg.CartLoad)
	if err != nil {
		return false, "", errors.New(errors.RegressionDigestError, err)
	}

	// list of state information. we'll either save this in the event of
	// newRegression being true; or we'll use it to compare to the entries in
	// the specified state file
	state := make([]string, 0, 1024)

	// add the starting state of the tv
	if reg.State {
		state = append(state, tv.String())
	}

	// display ticker for progress meter
	dur, _ := time.ParseDuration("1s")
	tck := time.NewTicker(dur)

	// run emulation
	err = vcs.RunForFrameCount(reg.NumFrames, func(frame int) (bool, error) {
		// display progress meter every 1 second
		select {
		case <-tck.C:
			output.Write([]byte(fmt.Sprintf("\r%s[%d/%d (%.1f%%)]", msg, frame, reg.NumFrames, 100*(float64(frame)/float64(reg.NumFrames)))))
		default:
		}

		// store tv state at every step
		if reg.State {
			state = append(state, tv.String())
		}

		return true, nil
	})

	if err != nil {
		return false, "", errors.New(errors.RegressionDigestError, err)
	}

	if newRegression {
		reg.digest = dig.Hash()

		if reg.State {
			// create a unique filename
			reg.stateFile, err = uniqueFilename("state", reg.CartLoad)
			if err != nil {
				return false, "", errors.New(errors.RegressionDigestError, err)
			}

			// check that the filename is unique
			nf, _ := os.Open(reg.stateFile)

			// no need to bother with returned error. nf tells us everything we
			// need
			if nf != nil {
				msg := fmt.Sprintf("state recording file already exists (%s)", reg.stateFile)
				return false, "", errors.New(errors.RegressionDigestError, msg)
			}
			nf.Close()

			// create new file
			nf, err = os.Create(reg.stateFile)
			if err != nil {
				msg := fmt.Sprintf("error creating state recording file: %s", err)
				return false, "", errors.New(errors.RegressionDigestError, msg)
			}
			defer nf.Close()

			for i := range state {
				s := fmt.Sprintf("%s\n", state[i])
				if n, err := nf.WriteString(s); err != nil || len(s) != n {
					msg := fmt.Sprintf("error writing state recording file: %s", err)
					return false, "", errors.New(errors.RegressionDigestError, msg)
				}
			}
		}

		return true, "", nil
	}

	// if we reach this point then this is a regression test (not adding a new
	// test)

	// compare new state tracking with recorded state tracking
	if reg.State {
		nf, err := os.Open(reg.stateFile)
		if err != nil {
			msg := fmt.Sprintf("old state recording file not present (%s)", reg.stateFile)
			return false, "", errors.New(errors.RegressionDigestError, msg)
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
