package regression

import (
	"bufio"
	"fmt"
	"gopher2600/database"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/hardware/memory"
	"gopher2600/performance/limiter"
	"gopher2600/setup"
	"gopher2600/television/renderers"
	"io"
	"os"
	"strconv"
	"strings"
)

const frameEntryID = "frame"

const (
	frameFieldCartName int = iota
	frameFieldCartFormat
	frameFieldTVtype
	frameFieldNumFrames
	frameFieldState
	frameFieldDigest
	frameFieldNotes
	numFrameFields
)

// FrameRegression is the simplest regression type. it works by running the
// emulation for N frames and the screen digest recorded at that point.
// regression tests pass if the screen digest after N frames matches the stored
// value.
type FrameRegression struct {
	CartLoad     memory.CartridgeLoader
	TVtype       string
	NumFrames    int
	State        bool
	stateFile    string
	Notes        string
	screenDigest string
}

func deserialiseFrameEntry(fields []string) (database.Entry, error) {
	reg := &FrameRegression{}

	// basic sanity check
	if len(fields) > numFrameFields {
		return nil, errors.New(errors.RegressionFrameError, "too many fields")
	}
	if len(fields) < numFrameFields {
		return nil, errors.New(errors.RegressionFrameError, "too few fields")
	}

	// string fields need no conversion
	reg.screenDigest = fields[frameFieldDigest]
	reg.CartLoad.Filename = fields[frameFieldCartName]
	reg.CartLoad.Format = fields[frameFieldCartFormat]
	reg.TVtype = fields[frameFieldTVtype]
	reg.Notes = fields[frameFieldNotes]

	var err error

	// convert number of frames field
	reg.NumFrames, err = strconv.Atoi(fields[frameFieldNumFrames])
	if err != nil {
		msg := fmt.Sprintf("invalid numFrames field [%s]", fields[frameFieldNumFrames])
		return nil, errors.New(errors.RegressionFrameError, msg)
	}

	// convert state field
	if fields[frameFieldState] != "" {
		reg.State = true
		reg.stateFile = fields[frameFieldState]
	}

	return reg, nil
}

// ID implements the database.Entry interface
func (reg FrameRegression) ID() string {
	return frameEntryID
}

// String implements the database.Entry interface
func (reg FrameRegression) String() string {
	s := strings.Builder{}
	stateFile := ""
	if reg.State {
		stateFile = "[with state]"
	}
	s.WriteString(fmt.Sprintf("[%s] %s [%s] frames=%d %s", reg.ID(), reg.CartLoad.ShortName(), reg.TVtype, reg.NumFrames, stateFile))
	if reg.Notes != "" {
		s.WriteString(fmt.Sprintf(" [%s]", reg.Notes))
	}
	return s.String()
}

// Serialise implements the database.Entry interface
func (reg *FrameRegression) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			reg.CartLoad.Filename,
			reg.CartLoad.Format,
			reg.TVtype,
			strconv.Itoa(reg.NumFrames),
			reg.stateFile,
			reg.screenDigest,
			reg.Notes,
		},
		nil
}

// CleanUp implements the database.Entry interface
func (reg FrameRegression) CleanUp() error {
	err := os.Remove(reg.stateFile)
	if _, ok := err.(*os.PathError); ok {
		return nil
	}
	return err
}

// regress implements the regression.Regressor interface
func (reg *FrameRegression) regress(newRegression bool, output io.Writer, msg string) (bool, error) {
	output.Write([]byte(msg))

	tv, err := renderers.NewDigestTV(reg.TVtype, nil)
	if err != nil {
		return false, errors.New(errors.RegressionFrameError, err)
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return false, errors.New(errors.RegressionFrameError, err)
	}

	err = setup.AttachCartridge(vcs, reg.CartLoad)
	if err != nil {
		return false, errors.New(errors.RegressionFrameError, err)
	}

	state := make([]string, 0, 1024)

	// display progress meter every 1 second
	limiter, err := limiter.NewFPSLimiter(1)
	if err != nil {
		return false, errors.New(errors.RegressionFrameError, err)
	}

	// run emulation
	err = vcs.RunForFrameCount(reg.NumFrames, func(frame int) (bool, error) {
		if limiter.HasWaited() {
			output.Write([]byte(fmt.Sprintf("\r%s[%d/%d (%.1f%%)]", msg, frame, reg.NumFrames, 100*(float64(frame)/float64(reg.NumFrames)))))
		}
		return true, nil
	})

	if err != nil {
		return false, errors.New(errors.RegressionFrameError, err)
	}

	if newRegression {
		reg.screenDigest = tv.String()

		if reg.State {
			// create a unique filename
			reg.stateFile = uniqueFilename(reg.CartLoad)

			// check that the filename is unique
			nf, _ := os.Open(reg.stateFile)

			// no need to bother with returned error. nf tells us everything we
			// need
			if nf != nil {
				msg := fmt.Sprintf("state recording file already exists (%s)", reg.stateFile)
				return false, errors.New(errors.RegressionFrameError, msg)
			}
			nf.Close()

			// create new file
			nf, err = os.Create(reg.stateFile)
			if err != nil {
				msg := fmt.Sprintf("error creating state recording file: %s", err)
				return false, errors.New(errors.RegressionFrameError, msg)
			}
			defer nf.Close()

			for i := range state {
				s := fmt.Sprintf("%s\n", state[i])
				if n, err := nf.WriteString(s); err != nil || len(s) != n {
					msg := fmt.Sprintf("error writing state recording file: %s", err)
					return false, errors.New(errors.RegressionFrameError, msg)
				}
			}
		}

		return true, nil
	}

	// if we reach this point then this is a regression test (not adding a new
	// test)

	// compare new state tracking with recorded state tracking
	if reg.State {
		nf, err := os.Open(reg.stateFile)
		if err != nil {
			msg := fmt.Sprintf("old state recording file not present (%s)", reg.stateFile)
			return false, errors.New(errors.RegressionFrameError, msg)
		}
		defer nf.Close()

		reader := bufio.NewReader(nf)

		for i := range state {
			s, _ := reader.ReadString('\n')
			s = strings.TrimRight(s, "\n")
			if s != state[i] {
				fmt.Println("\n", i, s, state[i])
				return false, nil
			}
		}
	}

	success := tv.String() == reg.screenDigest

	return success, nil
}
