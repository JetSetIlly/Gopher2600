package regression

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/recorder"
	"gopher2600/television/renderers"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	playbackFieldScript int = iota
	numPlaybackFields
)

// PlaybackRegression represents a regression type that processes a VCS
// recording. playback regressions can take a while to run because by their
// nature they extend over many frames - many more than is typical with the
// FrameRegression type.
type PlaybackRegression struct {
	key    int
	Script string
}

func (reg PlaybackRegression) getID() string {
	return "playback"
}

func newPlaybackRegression(key int, csv string) (*PlaybackRegression, error) {
	// loop through file until EOF is reached
	fields := strings.Split(csv, ",")

	reg := &PlaybackRegression{
		key:    key,
		Script: fields[playbackFieldScript],
	}

	return reg, nil
}

func (reg *PlaybackRegression) setKey(key int) {
	reg.key = key
}

func (reg PlaybackRegression) getKey() int {
	return reg.key
}

func (reg *PlaybackRegression) getCSV() string {
	return fmt.Sprintf("%s%s%s",
		csvLeader(reg), fieldSep,
		reg.Script,
	)
}

func (reg PlaybackRegression) String() string {
	return fmt.Sprintf("[%s] %s", reg.getID(), reg.Script)
}

func (reg *PlaybackRegression) regress(newRegression bool) (bool, error) {
	plb, err := recorder.NewPlayback(reg.Script)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionFail, err)
	}

	digest, err := renderers.NewDigestTV(plb.TVtype, nil)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionFail, err)
	}

	vcs, err := hardware.NewVCS(digest)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionFail, err)
	}

	err = vcs.AttachCartridge(plb.CartFile)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionFail, err)
	}

	err = plb.AttachToVCS(vcs)
	if err != nil {
		return false, errors.NewFormattedError(errors.RegressionFail, err)
	}

	err = vcs.Run(func() (bool, error) {
		// TODO: timeout option
		return true, nil
	})
	if err != nil {
		// the PowerOff error is expected. if we receive it then that means
		// the regression test has succeeded
		switch err := err.(type) {
		case errors.FormattedError:
			if err.Errno != errors.PowerOff {
				return false, errors.NewFormattedError(errors.RegressionFail, err)
			}
		default:
			return false, errors.NewFormattedError(errors.RegressionFail, err)
		}
	}

	if newRegression {
		// make sure regression script directory exists
		err = os.MkdirAll(regressionScripts, 0755)
		if err != nil {
			msg := fmt.Sprintf("cannot store playback script: %s", err)
			return false, errors.NewFormattedError(errors.RegressionDBError, msg)
		}

		// create a (hopefully) unique name for copied script file
		shortCartName := path.Base(plb.CartFile)
		shortCartName = strings.TrimSuffix(shortCartName, path.Ext(plb.CartFile))
		n := time.Now()
		timestamp := fmt.Sprintf("%04d%02d%02d_%02d%02d%02d", n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second())
		newScript := fmt.Sprintf("%s_%s", shortCartName, timestamp)
		newScript = filepath.Join(regressionScripts, newScript)

		// check that the filename is unique
		nf, err := os.Open(newScript)
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
		defer func() {
			nf.Close()
		}()

		// open old file
		of, err := os.Open(reg.Script)
		if err != nil {
			msg := fmt.Sprintf("error copying playback script: %s", err)
			return false, errors.NewFormattedError(errors.RegressionDBError, msg)
		}
		defer func() {
			of.Close()
		}()

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

func (reg PlaybackRegression) cleanUp() {
	// ignore errors from remove process
	_ = os.Remove(reg.Script)
}
