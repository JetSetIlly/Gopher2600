package regression

import (
	"fmt"
	"gopher2600/database"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/performance/limiter"
	"gopher2600/recorder"
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
	Script string
	Notes  string
}

func deserialisePlaybackEntry(fields []string) (database.Entry, error) {
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
func (reg PlaybackRegression) CleanUp() {
	// ignore errors from remove process
	_ = os.Remove(reg.Script)
}

// regress implements the regression.Regressor interface
func (reg *PlaybackRegression) regress(newRegression bool, output io.Writer, message string) (bool, error) {
	output.Write([]byte(message))

	plb, err := recorder.NewPlayback(reg.Script)
	if err != nil {
		return false, errors.New(errors.RegressionError, err)
	}

	digest, err := renderers.NewDigestTV(plb.TVtype, nil)
	if err != nil {
		return false, errors.New(errors.RegressionError, err)
	}

	vcs, err := hardware.NewVCS(digest)
	if err != nil {
		return false, errors.New(errors.RegressionError, err)
	}

	err = plb.AttachToVCS(vcs)
	if err != nil {
		return false, errors.New(errors.RegressionError, err)
	}

	// not using setup.AttachCartridge. if the playback was recorded with setup
	// changes the events will have been copied into the playback script and
	// will be applied that way
	err = vcs.AttachCartridge(plb.CartFile)
	if err != nil {
		return false, errors.New(errors.RegressionError, err)
	}

	// run emulation and display progress meter every 1 second
	limiter, err := limiter.NewFPSLimiter(1)
	if err != nil {
		return false, errors.New(errors.RegressionError, err)
	}
	err = vcs.Run(func() (bool, error) {
		if limiter.HasWaited() {
			output.Write([]byte(fmt.Sprintf("\r%s [%s]", message, plb)))
		}
		return true, nil
	})
	if err != nil {
		if !errors.IsAny(err) {
			return false, errors.New(errors.RegressionError, err)
		}

		switch err.(errors.AtariError).Errno {
		// the PowerOff error is expected. if we receive it then that means
		// the regression test has succeeded
		case errors.PowerOff:
			break

		// PlaybackHashError means that a screen digest somewhere in the
		// playback script did not work. filter error and return false to
		// indicate failure
		case errors.PlaybackHashError:
			return false, nil

		default:
			return false, errors.New(errors.RegressionError, err)
		}

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
			return false, errors.New(errors.RegressionPlaybackError, msg)
		}
		nf.Close()

		// create new file
		nf, err = os.Create(newScript)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, errors.New(errors.RegressionPlaybackError, msg)
		}
		defer nf.Close()

		// open old file
		of, err := os.Open(reg.Script)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, errors.New(errors.RegressionPlaybackError, msg)
		}
		defer of.Close()

		// copy old file to new file
		_, err = io.Copy(nf, of)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, errors.New(errors.RegressionPlaybackError, msg)
		}

		// update script name in regression type
		reg.Script = newScript
	}

	return true, nil
}
