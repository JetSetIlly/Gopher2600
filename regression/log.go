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
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/setup"
)

const logEntryType = "log"

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
	Cartridge string
	Mapping   string
	TVtype    string
	NumFrames int
	Notes     string
	digest    string
}

func deserialiseLogEntry(fields database.SerialisedEntry) (database.Entry, error) {
	reg := &LogRegression{}

	// basic sanity check
	if len(fields) > numLogFields {
		return nil, fmt.Errorf("log: too many fields")
	}
	if len(fields) < numLogFields {
		return nil, fmt.Errorf("log: too few fields")
	}

	reg.Cartridge = fields[videoFieldCartName]
	reg.Mapping = fields[videoFieldCartMapping]
	reg.TVtype = fields[logFieldTVtype]
	reg.digest = fields[logFieldDigest]
	reg.Notes = fields[logFieldNotes]

	var err error

	reg.NumFrames, err = strconv.Atoi(fields[logFieldNumFrames])
	if err != nil {
		msg := fmt.Sprintf("invalid numFrames field [%s]", fields[logFieldNumFrames])
		return nil, fmt.Errorf("log: %s", msg)
	}

	return reg, nil
}

// EntryType implements the database.Entry interface.
func (reg LogRegression) EntryType() string {
	return logEntryType
}

// Serialise implements the database.Entry interface.
func (reg *LogRegression) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			reg.Cartridge,
			reg.Mapping,
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

// String implements the regressions.Regressor interface
func (reg LogRegression) String() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("[%s] %s [%s] frames=%d", reg.EntryType(),
		cartridgeloader.NameFromFilename(reg.Cartridge),
		reg.TVtype, reg.NumFrames))
	if reg.Notes != "" {
		s.WriteString(fmt.Sprintf(" [%s]", reg.Notes))
	}
	return s.String()
}

// redux implements the regression.Regressor interface.
func (reg *LogRegression) redux(messages io.Writer, tag string) (Regressor, error) {
	old := *reg
	return &old, reg.regress(true, messages, tag)
}

// regress implements the regression.Regressor interface.
func (reg *LogRegression) regress(newRegression bool, messages io.Writer, tag string) error {
	// make sure logger is clear
	logger.Clear()

	// echoing log output to buffer
	logOutput := &strings.Builder{}
	logger.SetEcho(logOutput, false)

	// start output with message
	messages.Write([]byte(tag))

	// create headless television. we'll use this to initialise the digester
	tv, err := television.NewSimpleTelevision(reg.TVtype)
	if err != nil {
		return fmt.Errorf("log: %w", err)
	}
	defer tv.End()
	tv.SetFPSCap(false)

	// create VCS and attach cartridge
	vcs, err := hardware.NewVCS(environment.MainEmulation, tv, nil, nil)
	if err != nil {
		return fmt.Errorf("log: %w", err)
	}

	// we want the machine in a known state. the easiest way to do this is to
	// default the hardware preferences
	vcs.Env.Normalise()

	cartload, err := cartridgeloader.NewLoaderFromFilename(reg.Cartridge, reg.Mapping, "AUTO", nil)
	if err != nil {
		return fmt.Errorf("log: %w", err)
	}
	defer cartload.Close()

	err = setup.AttachCartridge(vcs, cartload, true)
	if err != nil {
		return fmt.Errorf("log: %w", err)
	}

	// display ticker for progress meter
	dur, _ := time.ParseDuration("1s")
	tck := time.NewTicker(dur)

	// run emulation
	err = vcs.RunForFrameCount(reg.NumFrames, func() (govern.State, error) {
		// if the CPU is in the KIL state then the test will never end normally
		if vcs.CPU.Killed {
			return govern.Ending, fmt.Errorf("CPU in KIL state")
		}

		// display progress meter every 1 second
		select {
		case <-tck.C:
			frame := vcs.TV.GetCoords().Frame
			messages.Write([]byte(fmt.Sprintf("\r%s [%d/%d (%.1f%%)]", tag, frame, reg.NumFrames, 100*(float64(frame)/float64(reg.NumFrames)))))
		default:
		}

		return govern.Running, nil
	})

	if err != nil {
		return fmt.Errorf("log: %w", err)
	}

	// get hash of log output
	hash := sha1.Sum([]byte(logOutput.String()))

	// note hash value if this is a new regression entry
	if newRegression {
		reg.digest = fmt.Sprintf("%x", hash)
	}

	// compare hashes from this run and the specimen run
	if reg.digest != fmt.Sprintf("%x", hash) {
		return fmt.Errorf("digest mismatch")
	}

	return nil
}
