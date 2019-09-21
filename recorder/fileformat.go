package recorder

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"os"
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
// # vcs_playback
// # <cartridge name>
// # <cartridge hash>
// # <tv type>

const (
	lineMagicString int = iota
	lineCartName
	lineCartFormat
	lineCartHash
	lineTVtype
	numHeaderLines
)

const magicString = "vcs_playback"

func (rec *Recorder) writeHeader() error {
	lines := make([]string, numHeaderLines)

	// add header information
	lines[lineMagicString] = magicString
	lines[lineCartName] = rec.vcs.Mem.Cart.Filename
	lines[lineCartFormat] = rec.vcs.Mem.Cart.RequestedFormat
	lines[lineCartHash] = rec.vcs.Mem.Cart.Hash
	lines[lineTVtype] = fmt.Sprintf("%v\n", rec.vcs.TV.GetSpec().ID)

	line := strings.Join(lines, "\n")

	n, err := io.WriteString(rec.output, line)

	if err != nil {
		rec.output.Close()
		return errors.New(errors.RecordingError, err)
	}

	if n != len(line) {
		rec.output.Close()
		return errors.New(errors.RecordingError, "output truncated")
	}

	return nil
}

func (plb *Playback) readHeader(lines []string) error {
	if lines[lineMagicString] != magicString {
		return errors.New(errors.PlaybackError, fmt.Sprintf("not a valid playback transcript (%s)", plb.transcript))
	}

	// read header
	plb.CartLoad.Filename = lines[lineCartName]
	plb.CartLoad.Format = lines[lineCartFormat]
	plb.CartLoad.Hash = lines[lineCartHash]
	plb.TVtype = lines[lineTVtype]

	return nil
}

// IsPlaybackFile returns true if file appears to be a playback file.
func IsPlaybackFile(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer func() { f.Close() }()

	b := make([]byte, len(magicString))
	n, err := f.Read(b)
	if n != len(magicString) || err != nil {
		return false
	}

	return string(b) == magicString
}
