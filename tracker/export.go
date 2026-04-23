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

package tracker

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
)

type ExportFormat int

const (
	ExportTIA ExportFormat = iota
)

func (h *History) Export(format ExportFormat, title string) error {
	switch format {
	case ExportTIA:
		return h.exportTIA(title)
	}
	panic("unknown tracker export format")
}

func (h *History) exportTIA(title string) error {
	if len(h.Entries) == 0 {
		return nil
	}

	type note struct {
		Row  int `json:"row"`
		Col  int `json:"col"`
		Len  int `json:"len"`
		AudC int `json:"audc"`
		Vol  int `json:"vol"`
	}

	type song struct {
		Version       int       `json:"verson"`
		Title         string    `json:"title"`
		FramesPerStep int       `json:"framesPerStep"`
		GridDiv       int       `json:"gridDiv"`
		Measures      int       `json:"measures"`
		DefaultVol    int       `json:"defaultVol"`
		Notes         [2][]note `json:"notes"`
	}

	// start and end of history
	startFrame := h.Entries[0].Coords.Frame
	endFrame := h.Entries[len(h.Entries)-1].Coords.Frame

	// song information
	output := song{
		Version:       1,
		Title:         title,
		FramesPerStep: 1,
		GridDiv:       1,
		Measures:      endFrame - startFrame,
		DefaultVol:    0,
	}

	// create note information for output
	var lastFrame [2]Entry

	for i, e := range h.Entries {
		if audio.CmpRegisters(e.Registers, lastFrame[e.Channel].Registers) {
			continue
		}

		// find duration of the note
		duration := -1
		for _, f := range h.Entries[i+1:] {
			if f.Channel == e.Channel {
				duration = f.Coords.Frame - e.Coords.Frame - 1
				break
			}
		}
		if duration == -1 {
			duration = endFrame - e.Coords.Frame
		}

		n := note{
			Row:  int(e.Registers.Freq & 0x1f),
			Col:  e.Coords.Frame - startFrame,
			Len:  duration,
			AudC: int(e.Registers.Control & 0x0f),
			Vol:  int(e.Registers.Volume & 0x0f),
		}

		output.Notes[e.Channel] = append(output.Notes[e.Channel], n)
		lastFrame[e.Channel] = e
	}

	// marshall and save file
	d, err := json.MarshalIndent(output, "", "\t")
	if err != nil {
		return fmt.Errorf("exportTIA: %w", err)
	}

	f, err := os.Create(fmt.Sprintf("%s.tia", title))
	if err != nil {
		return fmt.Errorf("exportTIA: %w", err)
	}
	defer f.Close()

	_, err = f.Write(d)
	if err != nil {
		return fmt.Errorf("exportTIA: %w", err)
	}

	return nil
}
