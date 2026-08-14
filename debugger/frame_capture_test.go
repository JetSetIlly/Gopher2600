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
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

func TestFrameCaptureSave(t *testing.T) {
	capture := newFrameCapture(nil)
	signals := make([]signal.SignalAttributes, specification.AbsoluteMaxClks)
	for i := range signals {
		signals[i] = signal.SignalAttributes{Index: signal.NoSignal, Color: signal.ZeroBlack}
	}

	pixel := specification.ClksHBlank
	signals[pixel] = signal.SignalAttributes{Index: pixel, Color: 0x46}
	if err := capture.SetPixels(signals, len(signals)-1); err != nil {
		t.Fatal(err)
	}

	frameInfo := frameinfo.NewCurrent(specification.SpecNTSC)
	frameInfo.FrameNum = 42
	frameInfo.VisibleTop = 0
	frameInfo.VisibleBottom = 0
	if err := capture.NewFrame(frameInfo); err != nil {
		t.Fatal(err)
	}
	want := frameInfo.Spec.GetColorScreen(signals, pixel, specification.ClksScanline)

	// Mutating the working frame must not change the completed image.
	signals[pixel].VBlank = true
	if err := capture.SetPixels(signals, len(signals)-1); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "frame.png")
	savedInfo, err := capture.save(path)
	if err != nil {
		t.Fatal(err)
	}
	if savedInfo.FrameNum != frameInfo.FrameNum {
		t.Fatalf("frame number = %d, want %d", savedInfo.FrameNum, frameInfo.FrameNum)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := img.Bounds().Size().X, specification.ClksVisible; got != want {
		t.Fatalf("image width = %d, want %d", got, want)
	}
	if got, want := img.Bounds().Size().Y, 1; got != want {
		t.Fatalf("image height = %d, want %d", got, want)
	}

	got := img.At(0, 0)
	if got != want {
		t.Fatalf("pixel = %v, want %v", got, want)
	}
}

func TestFrameCaptureSaveBeforeFrame(t *testing.T) {
	capture := newFrameCapture(nil)
	_, err := capture.save(filepath.Join(t.TempDir(), "frame.png"))
	if err == nil {
		t.Fatal("expected error before first completed frame")
	}
}

func TestScreenshotCommandTemplate(t *testing.T) {
	if err := debuggerCommands.Validate("SCREENSHOT frame.png"); err != nil {
		t.Fatal(err)
	}
}
