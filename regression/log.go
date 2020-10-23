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
	"crypto/sha1"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/setup"
)

const logEntryID = "log"

const (
	logFieldCartName int = iota
	logFieldCartMapping
	logFieldTVtype
	logFieldNumFrames
	logFieldDigest
	logFieldNotes
	numLogFields
)

// LogRegression runs for N frames and takes a digest of the log at the end of
// the run. Regression passes if the subsequent runs produce the same
// log/digest.
type LogRegression struct {
	CartLoad  cartridgeloader.Loader
	TVtype    string
	NumFrames int
	Notes     string
	digest    string
}

func deserialiseLogEntry(fields database.SerialisedEntry) (database.Entry, error) {
	reg := &LogRegression{}

	// basic sanity check
	if len(fields) > numLogFields {
		return nil, curated.Errorf("log: too many fields")
	}
	if len(fields) < numLogFields {
		return nil, curated.Errorf("log: too few fields")
	}

	// string fields need no conversion
	reg.CartLoad.Filename = fields[logFieldCartName]
	reg.CartLoad.Mapping = fields[logFieldCartMapping]
	reg.TVtype = fields[logFieldTVtype]
	reg.digest = fields[logFieldDigest]
	reg.Notes = fields[logFieldNotes]

	var err error

	// convert number of frames field
	reg.NumFrames, err = strconv.Atoi(fields[logFieldNumFrames])
	if err != nil {
		msg := fmt.Sprintf("invalid numFrames field [%s]", fields[logFieldNumFrames])
		return nil, curated.Errorf("log: %v", msg)
	}

	return reg, nil
}

// ID implements the database.Entry interface.
func (reg LogRegression) ID() string {
	return logEntryID
}

// String implements the database.Entry interface.
func (reg LogRegression) String() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("[%s] %s [%s] frames=%d", reg.ID(), reg.CartLoad.ShortName(), reg.TVtype, reg.NumFrames))
	if reg.Notes != "" {
		s.WriteString(fmt.Sprintf(" [%s]", reg.Notes))
	}
	return s.String()
}

// Serialise implements the database.Entry interface.
func (reg *LogRegression) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			reg.CartLoad.Filename,
			reg.CartLoad.Mapping,
			reg.TVtype,
			strconv.Itoa(reg.NumFrames),
			reg.digest,
			reg.Notes,
		},
		nil
}

// CleanUp implements the database.Entry interface.
func (reg LogRegression) CleanUp() error {
	return nil
}

// regress implements the regression.Regressor interface.
func (reg *LogRegression) regress(newRegression bool, output io.Writer, msg string, skipCheck func() bool) (bool, string, error) {
	// make sure logger is clear
	logger.Clear()

	output.Write([]byte(msg))

	// create headless television. we'll use this to initialise the digester
	tv, err := television.NewTelevision(reg.TVtype)
	if err != nil {
		return false, "", curated.Errorf("log: %v", err)
	}
	defer tv.End()

	// create VCS and attach cartridge
	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return false, "", curated.Errorf("log: %v", err)
	}

	// we want the machine in a known state. the easiest way to do this is to
	// reset the hardware preferences
	err = vcs.Prefs.Reset()
	if err != nil {
		return false, "", curated.Errorf("log: %v", err)
	}

	err = setup.AttachCartridge(vcs, reg.CartLoad)
	if err != nil {
		return false, "", curated.Errorf("log: %v", err)
	}

	// display ticker for progress meter
	dur, _ := time.ParseDuration("1s")
	tck := time.NewTicker(dur)

	// writing log output to buffer
	logOutput := &strings.Builder{}

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

		logger.WriteRecent(logOutput)

		return true, nil
	})

	if err != nil {
		return false, "", curated.Errorf("log: %v", err)
	}

	// get hash of log output
	hash := sha1.Sum([]byte(logOutput.String()))

	// note hash value if this is a new regression entry
	if newRegression {
		reg.digest = fmt.Sprintf("%x", hash)
	}

	// compare hashes from this run and the specimen run
	if reg.digest != fmt.Sprintf("%x", hash) {
		return false, "digest mismatch", nil
	}

	return true, "", nil
}
