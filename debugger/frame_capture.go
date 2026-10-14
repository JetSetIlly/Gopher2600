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

package debugger

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// frameCapture keeps an immutable copy of the last completed TV frame. Keeping
// a separate working image prevents a screenshot taken at a breakpoint from
// containing pixels from the next, incomplete frame.
type frameCapture struct {
	tv        *television.Television
	working   *image.RGBA
	completed *image.RGBA
	frameInfo frameinfo.Current
}

func newFrameCapture(tv *television.Television) *frameCapture {
	capture := &frameCapture{
		tv:        tv,
		working:   image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines)),
		frameInfo: frameinfo.NewCurrent(specification.SpecNTSC),
	}
	if tv != nil {
		capture.frameInfo = tv.GetFrameInfo()
	}
	capture.Reset()
	return capture
}

func (capture *frameCapture) NewFrame(frameInfo frameinfo.Current) error {
	crop := frameInfo.Crop().Intersect(capture.working.Bounds())
	if crop.Empty() {
		return fmt.Errorf("screenshot: invalid frame bounds %s", crop)
	}

	completed := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	draw.Draw(completed, completed.Bounds(), capture.working, crop.Min, draw.Src)

	capture.completed = completed
	capture.frameInfo = frameInfo
	return nil
}

func (capture *frameCapture) NewScanline(int) error {
	return nil
}

func (capture *frameCapture) SetPixels(sig []signal.SignalAttributes, last int) error {
	if capture.tv != nil {
		capture.frameInfo = capture.tv.GetFrameInfo()
	}

	limit := min(last+1, len(sig), len(capture.working.Pix)/4)
	for i := 0; i < len(capture.working.Pix)/4; i++ {
		col := color.RGBA{A: 255}
		if i < limit && !sig[i].VBlank && sig[i].Index != signal.NoSignal {
			col = capture.frameInfo.Spec.GetColorScreen(sig, i, specification.ClksScanline)
			col.A = 255
		}

		offset := i * 4
		capture.working.Pix[offset] = col.R
		capture.working.Pix[offset+1] = col.G
		capture.working.Pix[offset+2] = col.B
		capture.working.Pix[offset+3] = col.A
	}
	return nil
}

func (capture *frameCapture) Reset() {
	for i := 0; i < len(capture.working.Pix); i += 4 {
		capture.working.Pix[i] = 0
		capture.working.Pix[i+1] = 0
		capture.working.Pix[i+2] = 0
		capture.working.Pix[i+3] = 255
	}
	capture.completed = nil
}

func (capture *frameCapture) EndRendering() error {
	return nil
}

func (capture *frameCapture) save(path string) (frameinfo.Current, error) {
	if capture.completed == nil {
		return frameinfo.Current{}, fmt.Errorf("screenshot: no completed frame available")
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), ".gopher2600-screenshot-*.png")
	if err != nil {
		return frameinfo.Current{}, fmt.Errorf("screenshot: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := png.Encode(temporary, capture.completed); err != nil {
		_ = temporary.Close()
		return frameinfo.Current{}, fmt.Errorf("screenshot: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return frameinfo.Current{}, fmt.Errorf("screenshot: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return frameinfo.Current{}, fmt.Errorf("screenshot: %w", err)
	}

	return capture.frameInfo, nil
}
