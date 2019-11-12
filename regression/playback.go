package regression

import (
	"fmt"
	"gopher2600/database"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/performance/limiter"
	"gopher2600/recorder"
	"gopher2600/screendigest"
	"gopher2600/television"
	"io"
	"os"
	"path"
	"strings"
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

	tv, err := screendigest.NewSHA1(plb.TVtype, nil)
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

	// display progress meter every 1 second
	limiter, err := limiter.NewFPSLimiter(1)
	if err != nil {
		return false, "", errors.New(errors.RegressionPlaybackError, err)
	}

	// run emulation
	err = vcs.Run(func() (bool, error) {
		hasEnded, err := plb.EndFrame()
		if err != nil {
			return false, errors.New(errors.RegressionPlaybackError, err)
		}
		if hasEnded {
			return false, errors.New(errors.RegressionPlaybackError, "playback has not ended as expected")
		}

		if limiter.HasWaited() {
			output.Write([]byte(fmt.Sprintf("\r%s [%s]", msg, plb)))
		}
		return true, nil
	})

	if err != nil {
		if !errors.IsAny(err) {
			return false, "", errors.New(errors.RegressionPlaybackError, err)
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
		newScript := uniqueFilename("playback", plb.CartLoad)

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
