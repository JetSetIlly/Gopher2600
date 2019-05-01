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

// PlaybackRegression represents a regression type that processes vcs playback
// recording
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
		return false, err
	}

	digest, err := renderers.NewDigestTV(plb.TVtype, nil)
	if err != nil {
		return false, err
	}

	vcs, err := hardware.NewVCS(digest)
	if err != nil {
		return false, err
	}

	err = vcs.AttachCartridge(plb.CartFile)
	if err != nil {
		return false, err
	}

	err = plb.AttachToVCS(vcs)
	if err != nil {
		return false, err
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
				return false, err
			}
		default:
			return false, err
		}
	}

	if newRegression {
		// make sure regression script directory exists
		err = os.MkdirAll(regressionScripts, 0755)
		if err != nil {
			return false, err
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
			return false, errors.NewFormattedError(errors.RegressionFail, "cannot store playback file in regression database")
		}
		nf.Close()

		// create new file
		nf, err = os.Create(newScript)
		if err != nil {
			return false, err
		}
		defer func() {
			nf.Close()
		}()

		// open old file
		of, err := os.Open(reg.Script)
		if err != nil {
			return false, err
		}
		defer func() {
			of.Close()
		}()

		// copy old file to new file
		_, err = io.Copy(nf, of)
		if err != nil {
			return false, err
		}

		// update script name in regression type
		reg.Script = newScript
	}

	return true, nil
}

func (reg PlaybackRegression) cleanUp() {
	_ = os.Remove(reg.Script)
}
