package regression

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/performance/limiter"
	"gopher2600/recorder"
	"gopher2600/regression/database"
	"gopher2600/television/renderers"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
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
	key    int
	Script string
	Notes  string
}

func deserialisePlaybackEntry(key int, csv string) (database.Entry, error) {
	reg := &PlaybackRegression{key: key}

	fields := strings.Split(csv, ",")

	// basic sanity check
	if len(fields) > numPlaybackFields {
		return nil, errors.NewFormattedError(errors.RegressionDBError, "too many fields in frame playback entry")
	}
	if len(fields) < numPlaybackFields {
		return nil, errors.NewFormattedError(errors.RegressionDBError, "too few fields in frame playback entry")
	}

	// string fields need no conversion
	reg.Script = fields[playbackFieldScript]
	reg.Notes = fields[playbackFieldNotes]

	return reg, nil
}

// GetID implements the database.Entry interface
func (reg PlaybackRegression) GetID() string {
	return playbackEntryID
}

// SetKey implements the database.Entry interface
func (reg *PlaybackRegression) SetKey(key int) {
	reg.key = key
}

// GetKey implements the database.Entry interface
func (reg PlaybackRegression) GetKey() int {
	return reg.key
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
func (reg PlaybackRegression) CleanUp() {
	// ignore errors from remove process
	_ = os.Remove(reg.Script)
}

func (reg PlaybackRegression) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("[%s] %s", reg.GetID(), path.Base(reg.Script)))
	if reg.Notes != "" {
		s.WriteString(fmt.Sprintf(" [%s]", reg.Notes))
	}
	return s.String()
}

func (reg *PlaybackRegression) regress(newRegression bool, output io.Writer, message string) (bool, error) {
	output.Write([]byte(message))

	plb, err := recorder.NewPlayback(reg.Script)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionSetupError, err)
	}

	digest, err := renderers.NewDigestTV(plb.TVtype, nil)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionSetupError, err)
	}

	vcs, err := hardware.NewVCS(digest)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionSetupError, err)
	}

	err = vcs.AttachCartridge(plb.CartFile)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionSetupError, err)
	}

	err = plb.AttachToVCS(vcs)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionSetupError, err)
	}

	// run emulation and display progress meter every 1 second
	limiter, err := limiter.NewFPSLimiter(1)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionSetupError, err)
	}
	err = vcs.Run(func() (bool, error) {
		if limiter.HasWaited() {
			output.Write([]byte(fmt.Sprintf("\r%s [%s]", message, plb)))
		}
		return true, nil
	})
	if err != nil {
		switch err := err.(type) {
		case errors.FormattedError:
			switch err.Errno {
			// the PowerOff error is expected. if we receive it then that means
			// the regression test has succeeded
			case errors.PowerOff:
				break // switch

			// PlaybackHashError means that a screen digest somewhere in the
			// playback script did not work. filter error and return false to
			// indicate failure
			case errors.PlaybackHashError:
				return false, nil
			}
		}

		return false, errors.NewFormattedError(errors.RegressionSetupError, err)
	}

	// if this is a new regression we want to store the script in the
	// regressionScripts directory
	if newRegression {
		// create a (hopefully) unique name for copied script file
		shortCartName := path.Base(plb.CartFile)
		shortCartName = strings.TrimSuffix(shortCartName, path.Ext(plb.CartFile))
		n := time.Now()
		timestamp := fmt.Sprintf("%04d%02d%02d_%02d%02d%02d", n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second())
		newScript := fmt.Sprintf("%s_%s", shortCartName, timestamp)
		newScript = filepath.Join(regressionScripts, newScript)

		// check that the filename is unique
		nf, _ := os.Open(newScript)
		// no need to bother with returned error. nf tells us everything we
		// need
		if nf != nil {
			msg := fmt.Sprintf("script already exists (%s)", newScript)
			return false, errors.NewFormattedError(errors.RegressionDBError, msg)
		}
		nf.Close()

		// create new file
		nf, err = os.Create(newScript)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, errors.NewFormattedError(errors.RegressionDBError, msg)
		}
		defer nf.Close()

		// open old file
		of, err := os.Open(reg.Script)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, errors.NewFormattedError(errors.RegressionDBError, msg)
		}
		defer of.Close()

		// copy old file to new file
		_, err = io.Copy(nf, of)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, errors.NewFormattedError(errors.RegressionDBError, msg)
		}

		// update script name in regression type
		reg.Script = newScript
	}

	return true, nil
}
