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

package recorder

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
)

const (
	fieldID int = iota
	fieldEvent
	fieldEventData
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
// <magic string>
// <version string>
// <cartridge name>
// <cartridge hash>
// <tv type on startup>

const (
	lineMagicString int = iota
	lineVersion
	lineCartName
	lineCartHash
	lineTVSpec
	numHeaderLines
)

const magicString = "gopher2600playback"
const versionString = "1.0"

func (rec *Recorder) writeHeader() error {
	lines := make([]string, numHeaderLines)

	// add header information
	lines[lineMagicString] = magicString
	lines[lineVersion] = versionString
	lines[lineCartName] = rec.vcs.Mem.Cart.Filename
	lines[lineCartHash] = rec.vcs.Mem.Cart.Hash
	lines[lineTVSpec] = fmt.Sprintf("%v\n", rec.vcs.TV.GetReqSpecID())

	line := strings.Join(lines, "\n")

	n, err := io.WriteString(rec.output, line)

	if err != nil {
		rec.output.Close()
		return curated.Errorf("recorder: %v", err)
	}

	if n != len(line) {
		rec.output.Close()
		return curated.Errorf("recorder: output truncated")
	}

	return nil
}

func (plb *Playback) readHeader(lines []string) error {
	if lines[lineMagicString] != magicString {
		return curated.Errorf("playback: not a valid transcript (%s)", plb.transcript)
	}

	// read header
	plb.CartLoad.Filename = lines[lineCartName]
	plb.CartLoad.Hash = lines[lineCartHash]
	plb.TVSpec = lines[lineTVSpec]

	return nil
}

// IsPlaybackFile returns true if the specified file appears to be a playback
// file. It does not care about the nature of any errors that may be generated
// or if the file appears to be a playback file but is of an unsupported
// version.
func IsPlaybackFile(filename string) bool {
	// !!TODO: more nuanced results from IsPlaybackFile()

	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer func() { f.Close() }()

	// magic string verification
	b := make([]byte, len(magicString)+1)
	n, err := f.Read(b)
	if n != len(magicString)+1 || err != nil {
		return false
	}
	if string(b) != magicString+"\n" {
		return false
	}

	// version number verification
	b = make([]byte, len(versionString)+1)
	n, err = f.Read(b)
	if n != len(versionString)+1 || err != nil {
		return false
	}
	if string(b) != versionString+"\n" {
		return false
	}

	return true
}
