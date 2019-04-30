package recorder

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"strings"
)

const (
	fieldID int = iota
	fieldEvent
	fieldFrame
	fieldScanline
	fieldHorizPos
	fieldHash
	numFields
)

const fieldSep = ", "

// playback file header format
// ---------------------------
//
// # <cartridge name>
// # <cartridge hash>
// # <tv type>

const (
	lineCartName int = iota
	lineCartHash
	lineTVtype
	numHeaderLines
)

func (rec *Recorder) writeHeader() error {
	lines := make([]string, numHeaderLines)

	// add header information
	lines[lineCartName] = rec.vcs.Mem.Cart.Filename
	lines[lineCartHash] = rec.vcs.Mem.Cart.Hash
	lines[lineTVtype] = fmt.Sprintf("%v\n", rec.vcs.TV.GetSpec().ID)

	line := strings.Join(lines, "\n")

	n, err := io.WriteString(rec.output, line)

	if err != nil {
		rec.output.Close()
		return errors.NewFormattedError(errors.RecordingError, err)
	}

	if n != len(line) {
		rec.output.Close()
		return errors.NewFormattedError(errors.RecordingError, "output truncated")
	}

	return nil
}

func (plb *Playback) readHeader(lines []string) error {
	// read header
	plb.CartName = lines[lineCartName]
	plb.CartHash = lines[lineCartHash]
	plb.TVtype = lines[lineTVtype]

	// validate header
	tvspec := plb.vcs.TV.GetSpec()
	if tvspec.ID != lines[lineTVtype] {
		return errors.NewFormattedError(errors.PlaybackError, "current TV type does not match that in the recording")
	}

	return nil
}
