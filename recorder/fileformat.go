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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
)

const (
	fieldPortID int = iota
	fieldEvent
	fieldEventData
	fieldFrame
	fieldScanline
	fieldClock
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
// <tv type on startup>.

const (
	lineMagicString int = iota
	lineVersion
	lineCartName
	lineCartHash
	lineTVSpec
	numHeaderLines
)

// NOTE: playback scripts might fail if auto-controller method has changed (for
// example, the sensitivity values have changed)
//
// !!TODO: consider versioning the auto-controller and noting the version number in playback script

const magicString = "gopher2600playback"
const versionMajor = "1"
const versionMinor = "2"

// version history
// v1.0 original version
// v1.1 EventData for stick events extended
//		- compatibility code in controllers.Stick.HandleEvent()
// v1.2 fieldPortID (renamed from fieldID) stores new PortID values
//		- compatibility code in recorder.NewPlayback()

func version() string {
	return fmt.Sprintf("%s.%s", versionMajor, versionMinor)
}

func (rec *Recorder) writeHeader() error {
	lines := make([]string, numHeaderLines)

	// add header information
	lines[lineMagicString] = magicString
	lines[lineVersion] = version()
	lines[lineCartName] = rec.vcs.Mem.Cart.Filename
	lines[lineCartHash] = rec.vcs.Mem.Cart.Hash
	lines[lineTVSpec] = fmt.Sprintf("%v\n", rec.vcs.TV.GetReqSpecID())

	line := strings.Join(lines, "\n")

	n, err := io.WriteString(rec.output, line)

	if err != nil {
		rec.output.Close()
		return fmt.Errorf("recorder: %w", err)
	}

	if n != len(line) {
		rec.output.Close()
		return fmt.Errorf("recorder: output truncated")
	}

	return nil
}

func (plb *Playback) readHeader(lines []string, checkROM bool) error {
	if lines[lineMagicString] != magicString {
		return fmt.Errorf("playback: not a valid transcript (%s)", plb.transcript)
	}

	var err error

	// read header
	plb.CartLoad, err = cartridgeloader.NewLoaderFromFilename(lines[lineCartName], "AUTO")
	if err != nil {
		return fmt.Errorf("playback: %w", err)
	}

	if checkROM {
		plb.CartLoad.HashSHA1 = lines[lineCartHash]
	}

	plb.TVSpec = lines[lineTVSpec]

	return nil
}

// sentinal errors for IsPlaybackFile()
var NotAPlaybackFile = errors.New("not a playback file")
var UnsupportedVersion = errors.New("unsupported version")

// IsPlaybackFile return nil if file is a playback file and is a supported
// version. If file is not a playback file then the sentinal error
// NotAPlaybackFile is returned.
//
// Recognised playback files but where the version is unspported will result in
// an UnsupportedVersion error.
func IsPlaybackFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("%w: %w", NotAPlaybackFile, err)
	}
	defer func() { f.Close() }()

	reader := bufio.NewReader(f)

	// magic string comparison
	m, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("%w: %w", NotAPlaybackFile, err)
	}
	m = strings.TrimSuffix(m, "\n")
	if m != magicString {
		return fmt.Errorf("%w: unrecognised format", NotAPlaybackFile)
	}

	// version string comparison
	v, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("%w: %w", NotAPlaybackFile, err)
	}
	v = strings.TrimSuffix(v, "\n")

	// split into major/minor numbers
	versionParts := strings.Split(v, ".")
	if len(versionParts) != 2 {
		return fmt.Errorf("%w: unrecognised format", NotAPlaybackFile)
	}

	// major versions must match
	if versionMajor != versionParts[0] {
		return fmt.Errorf("%w: %s", UnsupportedVersion, v)
	}

	// earlier minor versions are supported
	if versionMinor < versionParts[1] {
		return fmt.Errorf("%w: %s", UnsupportedVersion, v)
	}

	return nil
}
